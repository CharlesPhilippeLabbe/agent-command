package ops

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"gopkg.in/yaml.v2"
)

var noteStatuses = []string{"open", "resolved"}

// Note is one commander note: a file in an operation's notes/ directory,
// anchored to a block of a plan, order or report by the block's source lines
// (the quote). Line is only a hint for finding the quote again.
type Note struct {
	Op      string `yaml:"op" json:"op"`
	Target  string `yaml:"target" json:"target"` // relative to the op dir
	Line    int    `yaml:"line" json:"line"`
	Status  string `yaml:"status" json:"status"`
	Created string `yaml:"created" json:"created"`

	Quote      string `yaml:"-" json:"quote"`
	Text       string `yaml:"-" json:"text"`
	Resolution string `yaml:"-" json:"resolution"`

	Path     string `yaml:"-" json:"path"`     // served path of the note file
	Detached bool   `yaml:"-" json:"detached"` // quote not found in the target
}

// ParseNote reads a note file: frontmatter, then the QUOTE, NOTE and
// optional RESOLUTION sections.
func ParseNote(src []byte) (*Note, error) {
	fm, body, ok := split(src)
	if !ok {
		return nil, errNoFrontmatter
	}
	n := &Note{}
	if err := yaml.UnmarshalStrict(fm, n); err != nil {
		return nil, err
	}
	if !slices.Contains(noteStatuses, n.Status) {
		return nil, fmt.Errorf("unknown status %q", n.Status)
	}

	lines := strings.Split(string(body), "\n")
	q := slices.Index(lines, "# QUOTE")
	if q < 0 || q+1 >= len(lines) {
		return nil, errors.New("no QUOTE")
	}
	fence := lines[q+1]
	if len(fence) < 3 || strings.Trim(fence, "`") != "" {
		return nil, errors.New("QUOTE not fenced")
	}
	end := slices.Index(lines[q+2:], fence)
	if end < 0 {
		return nil, errors.New("QUOTE fence not closed")
	}
	n.Quote = strings.Join(lines[q+2:q+2+end], "\n")

	rest := lines[q+3+end:]
	t := slices.Index(rest, "# NOTE")
	if t < 0 {
		return nil, errors.New("no NOTE")
	}
	rest = rest[t+1:]
	if r := slices.Index(rest, "# RESOLUTION"); r >= 0 {
		n.Resolution = strings.TrimSpace(strings.Join(rest[r+1:], "\n"))
		rest = rest[:r]
	}
	n.Text = strings.TrimSpace(strings.Join(rest, "\n"))
	return n, nil
}

// FormatNote writes n as a note file. ParseNote reads it back unchanged.
func FormatNote(n *Note) []byte {
	fm, _ := yaml.Marshal(n)
	fence := strings.Repeat("`", max(3, longestRun(n.Quote, '`')+1))
	var b bytes.Buffer
	fmt.Fprintf(&b, "---\n%s---\n# QUOTE\n%s\n%s\n%s\n\n# NOTE\n%s\n", fm, fence, n.Quote, fence, n.Text)
	if n.Resolution != "" {
		fmt.Fprintf(&b, "\n# RESOLUTION\n%s\n", n.Resolution)
	}
	return b.Bytes()
}

// longestRun is the length of the longest run of c in s.
func longestRun(s string, c byte) int {
	longest, run := 0, 0
	for i := range len(s) {
		if s[i] == c {
			run++
			longest = max(longest, run)
		} else {
			run = 0
		}
	}
	return longest
}

// NoteName is the file name of a note on target: "orders/03-api.md" with
// hex "3fa9" gives "03-api.3fa9.md".
func NoteName(target, hex string) string {
	return strings.TrimSuffix(path.Base(target), ".md") + "." + hex + ".md"
}

// Quote returns lines line to end of src, 1-based and frontmatter included.
func Quote(src []byte, line, end int) (string, error) {
	lines := fileLines(src)
	if line < 1 || end < line || end > len(lines) {
		return "", fmt.Errorf("lines %d-%d outside 1-%d", line, end, len(lines))
	}
	return strings.Join(lines[line-1:end], "\n"), nil
}

// BodyLine is the 1-based line of src where the body starts, after the
// frontmatter SplitFrontmatter cuts off; 1 without frontmatter.
func BodyLine(src []byte) int {
	_, body, ok := split(src)
	if !ok {
		return 1
	}
	norm := bytes.ReplaceAll(src, []byte("\r\n"), []byte("\n"))
	return bytes.Count(norm[:len(norm)-len(body)], []byte("\n")) + 1
}

// fileLines splits src into lines, "\r\n" read as "\n", without the empty
// line after a final newline.
func fileLines(src []byte) []string {
	s := strings.ReplaceAll(string(src), "\r\n", "\n")
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

// anchor finds n's quote in src: the match nearest n.Line, the earlier one
// on a tie. Found, n.Line moves to it; not found, n is detached.
func (n *Note) anchor(src []byte) {
	lines := fileLines(src)
	quote := strings.Split(n.Quote, "\n")
	best, bestDist := 0, 0
	for i := 0; i+len(quote) <= len(lines); i++ {
		if !slices.Equal(lines[i:i+len(quote)], quote) {
			continue
		}
		if dist := abs(i + 1 - n.Line); best == 0 || dist < bestDist {
			best, bestDist = i+1, dist
		}
	}
	if best == 0 {
		n.Detached = true
		return
	}
	n.Line = best
}

func abs(x int) int { return max(x, -x) }

// Notes lists the open and the resolved notes on target (relative to opDir),
// each anchored in src, the target's current content. Files that don't parse
// are skipped. Both lists are ordered by created, then file name.
func Notes(fsys fs.FS, opDir, target string, src []byte) (open, resolved []*Note) {
	open, resolved = []*Note{}, []*Note{}
	for _, p := range mdFiles(fsys, path.Join(opDir, "notes")) {
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			continue
		}
		n, err := ParseNote(data)
		if err != nil || n.Target != target {
			continue
		}
		n.Path = p
		n.anchor(src)
		if n.Status == "open" {
			open = append(open, n)
		} else {
			resolved = append(resolved, n)
		}
	}
	byCreated := func(a, b *Note) int {
		return cmp.Or(cmp.Compare(a.Created, b.Created), cmp.Compare(a.Path, b.Path))
	}
	slices.SortStableFunc(open, byCreated)
	slices.SortStableFunc(resolved, byCreated)
	return open, resolved
}
