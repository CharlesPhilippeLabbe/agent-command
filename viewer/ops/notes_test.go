package ops

import (
	"reflect"
	"testing"
	"testing/fstest"
)

const noteSrc = "---\nop: vacuum-robot\ntarget: orders/02-api.md\nline: 9\nstatus: open\ncreated: 2026-11-07 14:03\n---\n# QUOTE\n```\n- RISK BODY\n```\n\n# NOTE\nToo vague.\n"

func TestParseNote(t *testing.T) {
	n, err := ParseNote([]byte(noteSrc))
	if err != nil {
		t.Fatal(err)
	}
	want := &Note{Op: "vacuum-robot", Target: "orders/02-api.md", Line: 9, Status: "open", Created: "2026-11-07 14:03", Quote: "- RISK BODY", Text: "Too vague."}
	if !reflect.DeepEqual(n, want) {
		t.Errorf("ParseNote = %+v, want %+v", n, want)
	}

	bad := map[string]string{
		"no NOTE":        "---\nop: x\ntarget: plan.md\nline: 1\nstatus: open\ncreated: x\n---\n# QUOTE\n```\nq\n```\n",
		"unknown status": "---\nop: x\ntarget: plan.md\nline: 1\nstatus: done\ncreated: x\n---\n# QUOTE\n```\nq\n```\n# NOTE\nn\n",
		"no frontmatter": "# QUOTE\n```\nq\n```\n# NOTE\nn\n",
	}
	for name, src := range bad {
		if _, err := ParseNote([]byte(src)); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestFormatNoteRoundTrip(t *testing.T) {
	n := &Note{
		Op: "vacuum-robot", Target: "plan.md", Line: 4, Status: "resolved", Created: "2026-11-07 14:03",
		Quote:      "Run:\n\n```sh\ngo test ./...\n```",
		Text:       "Say which package.\n\n# not a section",
		Resolution: "- plan.md: D4 names the package.",
	}
	got, err := ParseNote(FormatNote(n))
	if err != nil {
		t.Fatalf("%v\n%s", err, FormatNote(n))
	}
	if !reflect.DeepEqual(got, n) {
		t.Errorf("round trip = %+v, want %+v\n%s", got, n, FormatNote(n))
	}
}

func TestNoteName(t *testing.T) {
	for target, want := range map[string]string{
		"plan.md":             "plan.3fa9.md",
		"orders/03-api.md":    "03-api.3fa9.md",
		"reports/03-api.2.md": "03-api.2.3fa9.md",
	} {
		if got := NoteName(target, "3fa9"); got != want {
			t.Errorf("NoteName(%q) = %q, want %q", target, got, want)
		}
	}
}

func TestQuote(t *testing.T) {
	src := []byte("---\nop: x\n---\n# GOAL\nline five\nline six\n")
	got, err := Quote(src, 5, 6)
	if err != nil || got != "line five\nline six" {
		t.Errorf("Quote(5, 6) = %q, %v", got, err)
	}
	for _, r := range [][2]int{{6, 7}, {0, 1}, {5, 4}} {
		if _, err := Quote(src, r[0], r[1]); err == nil {
			t.Errorf("Quote(%d, %d): no error", r[0], r[1])
		}
	}
}

func TestBodyLine(t *testing.T) {
	for src, want := range map[string]int{
		"---\nop: x\nstatus: y\n---\n# GOAL\n": 5,
		"---\r\nop: x\r\n---\r\n# GOAL\r\n":    4,
		"# GOAL\n":                             1,
	} {
		if got := BodyLine([]byte(src)); got != want {
			t.Errorf("BodyLine(%q) = %d, want %d", src, got, want)
		}
	}
}

func TestAnchor(t *testing.T) {
	src := "a\nB\nc\nd\nB\nf\ng\nB\n"
	cases := []struct {
		name     string
		quote    string
		line     int
		src      string
		want     int
		detached bool
	}{
		{"same line", "c\nd", 3, src, 3, false},
		{"moved down", "c\nd", 3, "new\nnew\n" + src, 5, false},
		{"duplicate picks nearest", "B", 7, src, 8, false},
		{"tie picks earlier", "B", 3, "a\nB\nc\nB\n", 2, false},
		{"edited quote", "c\nD", 3, src, 3, true},
	}
	for _, c := range cases {
		n := &Note{Quote: c.quote, Line: c.line}
		n.anchor([]byte(c.src))
		if n.Line != c.want || n.Detached != c.detached {
			t.Errorf("%s: line %d detached %v, want %d %v", c.name, n.Line, n.Detached, c.want, c.detached)
		}
	}
}

func TestNotes(t *testing.T) {
	note := func(target, status, created string) *fstest.MapFile {
		return file("---\nop: vacuum-robot\ntarget: " + target + "\nline: 2\nstatus: " + status + "\ncreated: " + created + "\n---\n# QUOTE\n```\nPLAN BODY\n```\n\n# NOTE\nn\n")
	}
	fsys := fstest.MapFS{
		dir + "/notes/plan.bbbb.md":    note("plan.md", "open", "2026-11-07 10:00"),
		dir + "/notes/plan.aaaa.md":    note("plan.md", "open", "2026-11-07 10:00"),
		dir + "/notes/plan.0000.md":    note("plan.md", "open", "2026-11-07 11:00"),
		dir + "/notes/plan.cccc.md":    note("plan.md", "resolved", "2026-11-07 09:00"),
		dir + "/notes/01-base.dddd.md": note("orders/01-base.md", "open", "2026-11-07 09:00"),
		dir + "/notes/plan.eeee.md":    file("not a note\n"),
	}
	src := []byte("---\nop: vacuum-robot\n---\n# GOAL\nPLAN BODY\n")
	got, resolved := Notes(fsys, dir, "plan.md", src)
	var paths []string
	for _, n := range got {
		paths = append(paths, n.Path)
		if n.Line != 5 || n.Detached {
			t.Errorf("%s: line %d detached %v, want 5 false", n.Path, n.Line, n.Detached)
		}
	}
	want := []string{dir + "/notes/plan.aaaa.md", dir + "/notes/plan.bbbb.md", dir + "/notes/plan.0000.md"}
	if !reflect.DeepEqual(paths, want) {
		t.Errorf("open notes = %v, want %v", paths, want)
	}
	if len(resolved) != 1 || resolved[0].Path != dir+"/notes/plan.cccc.md" {
		t.Errorf("resolved notes = %+v, want only plan.cccc.md", resolved)
	}

	if none, _ := Notes(fsys, dir, "orders/02-api.md", src); none == nil || len(none) != 0 {
		t.Errorf("open notes with none = %#v, want empty non-nil", none)
	}
}
