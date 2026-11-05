// Package ops reads .saved/ops/<op>/ operations (plan, orders, reports) from
// their YAML frontmatter and derives order status and dependency state.
// Only frontmatter is kept: bodies are read and discarded.
package ops

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

// skipDirs are never walked into: too big, and never hold operations.
var skipDirs = map[string]bool{".git": true, "node_modules": true, ".venv": true}

// Skipped reports whether a directory of this name is never walked into.
func Skipped(name string) bool { return skipDirs[name] }

var (
	planStatuses   = []string{"approved", "issued", "active", "closed"}
	reportStatuses = []string{"in-progress", "done", "partial", "blocked", "refused", "dropped"}
	ranks          = []string{"soldier", "captain", "major"}
	risks          = []string{"low", "medium", "high"}
)

// Num is an order number. Decoded from its literal text so "08" is 8, not an
// invalid octal.
type Num int

func (n *Num) UnmarshalYAML(unmarshal func(any) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("order number %q: %w", s, err)
	}
	*n = Num(v)
	return nil
}

type Operation struct {
	Project  string   `json:"project"`
	Name     string   `json:"name"`
	Dir      string   `json:"dir"`
	Plan     Plan     `json:"plan"`
	Orders   []*Order `json:"orders"`
	Problems []string `json:"problems"`

	Columns int `json:"columns"` // dependency depth of the deepest order
	Done    int `json:"done"`
	Dropped int `json:"dropped"`
	Total   int `json:"total"`
}

type Plan struct {
	Op      string `yaml:"op" json:"op"`
	Status  string `yaml:"status" json:"status"`
	Created string `yaml:"created" json:"created"`
	Path    string `yaml:"-" json:"path"`
}

type Order struct {
	Op      string `yaml:"op" json:"op"`
	Order   Num    `yaml:"order" json:"order"`
	Slug    string `yaml:"slug" json:"slug"`
	Rank    string `yaml:"rank" json:"rank"`
	Risk    string `yaml:"risk" json:"risk"` // optional
	Rev     int    `yaml:"rev" json:"rev"`
	Depends []Num  `yaml:"depends" json:"depends"`

	Path    string    `yaml:"-" json:"path"`
	NN      string    `yaml:"-" json:"nn"`
	Reports []*Report `yaml:"-" json:"reports"`
	Status  string    `yaml:"-" json:"status"`  // latest report's, else "pending"
	Waiting []Num     `yaml:"-" json:"waiting"` // dependencies not done yet
	Stale   bool      `yaml:"-" json:"stale"`   // revised since the latest report
	Column  int       `yaml:"-" json:"column"`  // 1 + depth in the dependency graph
}

type Report struct {
	Op       string `yaml:"op" json:"op"`
	Order    Num    `yaml:"order" json:"order"`
	Slug     string `yaml:"slug" json:"slug"`
	Attempt  int    `yaml:"attempt" json:"attempt"`
	OrderRev int    `yaml:"order_rev" json:"orderRev"`
	OrderSha string `yaml:"order_sha" json:"orderSha"`
	Rank     string `yaml:"rank" json:"rank"`
	Status   string `yaml:"status" json:"status"`
	Greenlit string `yaml:"greenlit" json:"greenlit"`
	Date     string `yaml:"date" json:"date"`
	Path     string `yaml:"-" json:"path"`
}

// SplitFrontmatter separates a leading "---" YAML block from the markdown body.
// Without frontmatter, meta is nil and body is src.
func SplitFrontmatter(src []byte) (meta yaml.MapSlice, body []byte, err error) {
	fm, body, ok := split(src)
	if !ok {
		return nil, src, nil
	}
	err = yaml.Unmarshal(fm, &meta)
	return meta, body, err
}

func split(src []byte) (fm, body []byte, ok bool) {
	src = bytes.ReplaceAll(src, []byte("\r\n"), []byte("\n"))
	rest, found := bytes.CutPrefix(src, []byte("---\n"))
	if !found {
		return nil, src, false
	}
	// The closing "---" line. A newline in front lets it be the first line.
	nl := append([]byte{'\n'}, rest...)
	if i := bytes.Index(nl, []byte("\n---\n")); i >= 0 {
		return rest[:i], rest[i+4:], true
	}
	if bytes.HasSuffix(nl, []byte("\n---")) {
		return rest[:len(rest)-3], nil, true
	}
	return nil, src, false
}

// Walk calls fn with every file under fsys in lexical order, skipping
// skipDirs and anything unreadable.
func Walk(fsys fs.FS, fn func(p string)) {
	fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
		case !d.IsDir():
			fn(p)
		case Skipped(d.Name()):
			return fs.SkipDir
		}
		return nil
	})
}

// Discover loads every operation found at **/.saved/ops/<op>/plan.md.
func Discover(fsys fs.FS) []*Operation {
	out := []*Operation{}
	Walk(fsys, func(p string) {
		if isPlan(p) {
			out = append(out, Load(fsys, path.Dir(p)))
		}
	})
	return out
}

func isPlan(p string) bool {
	opsDir := path.Dir(path.Dir(p))
	return path.Base(p) == "plan.md" && (opsDir == ".saved/ops" || strings.HasSuffix(opsDir, "/.saved/ops"))
}

// Load reads one operation directory. Format problems are collected, not fatal.
func Load(fsys fs.FS, dir string) *Operation {
	l := &loader{
		fsys: fsys,
		op: &Operation{
			Project: path.Dir(path.Dir(path.Dir(dir))), // <project>/.saved/ops/<op>, "." at the root
			Name:    path.Base(dir),
			Dir:     dir,
		},
		byNum: map[Num]*Order{},
		state: map[*Order]placement{},
	}
	l.loadPlan()
	l.loadOrders()
	l.loadReports()
	l.layout()
	return l.op
}

// loader builds one Operation and collects its problems.
type loader struct {
	fsys  fs.FS
	op    *Operation
	byNum map[Num]*Order

	// layout state: see place
	state map[*Order]placement
	trail []*Order // orders being placed, outermost first
	cycle string   // first dependency cycle found
}

type placement int

const (
	unplaced placement = iota
	placing
	placed
)

func (l *loader) problem(format string, a ...any) {
	l.op.Problems = append(l.op.Problems, fmt.Sprintf(format, a...))
}

// problems records each message against the file at p.
func (l *loader) problems(p string, msgs []string) {
	for _, m := range msgs {
		l.problem("%s: %s", path.Base(p), m)
	}
}

// read decodes the frontmatter of p into v, recording what is wrong with it;
// false when there is no frontmatter to use.
func (l *loader) read(p string, v any) bool {
	err := decode(l.fsys, p, v)
	if err != nil {
		l.problem("%s: %v", path.Base(p), err)
	}
	return err != errNoFrontmatter
}

func (l *loader) loadPlan() {
	pl := &l.op.Plan
	pl.Path = path.Join(l.op.Dir, "plan.md")
	l.read(pl.Path, pl)
	l.problems(pl.Path, pl.check(l.op.Name))
}

func (l *loader) loadOrders() {
	for _, p := range mdFiles(l.fsys, path.Join(l.op.Dir, "orders")) {
		o := &Order{Path: p}
		if !l.read(p, o) {
			continue
		}
		o.NN = fmt.Sprintf("%02d", o.Order)
		l.problems(p, o.check(l.op.Name))
		if prev, dup := l.byNum[o.Order]; dup {
			l.problem("%s: order %s already used by %s", path.Base(p), o.NN, path.Base(prev.Path))
			continue
		}
		l.byNum[o.Order] = o
		l.op.Orders = append(l.op.Orders, o)
	}
	slices.SortFunc(l.op.Orders, func(a, b *Order) int { return cmp.Compare(a.Order, b.Order) })
}

func (l *loader) loadReports() {
	for _, p := range mdFiles(l.fsys, path.Join(l.op.Dir, "reports")) {
		r := &Report{Path: p}
		if !l.read(p, r) {
			continue
		}
		l.problems(p, r.check())
		o, ok := l.byNum[r.Order]
		if !ok {
			l.problem("%s: no order %02d", path.Base(p), r.Order)
			continue
		}
		o.Reports = append(o.Reports, r)
	}
}

// layout places every order and counts the operation's progress.
func (l *loader) layout() {
	for _, o := range l.op.Orders {
		l.place(o)
		l.op.Columns = max(l.op.Columns, o.Column)
		switch o.Status {
		case "done":
			l.op.Done++
		case "dropped":
			l.op.Dropped++
		}
	}
	l.op.Total = len(l.op.Orders)
	if l.cycle != "" {
		l.problem("dependency cycle: %s", l.cycle)
	}
}

// place settles o after its dependencies, so their status and column are
// final when o reads them. Each order is placed once.
func (l *loader) place(o *Order) {
	if l.state[o] != unplaced {
		return
	}
	l.state[o] = placing
	l.trail = append(l.trail, o)

	l.problems(o.Path, o.settle())
	o.Column = 1
	for _, d := range o.Depends {
		dep, ok := l.byNum[d]
		if !ok {
			l.problem("%s: depends on missing order %02d", path.Base(o.Path), d)
			continue
		}
		l.follow(dep)
		o.Column = max(o.Column, dep.Column+1)
		if dep.Status != "done" && o.Status != "dropped" {
			o.Waiting = append(o.Waiting, d)
		}
	}

	l.trail = l.trail[:len(l.trail)-1]
	l.state[o] = placed
}

// follow places a dependency not placed yet. One still being placed is on the
// trail, so the dependencies have come back to it: a cycle. The first one is
// recorded; the dependency is otherwise used as it is.
func (l *loader) follow(dep *Order) {
	switch l.state[dep] {
	case unplaced:
		l.place(dep)
	case placing:
		if l.cycle == "" {
			l.cycle = cycleText(l.trail[slices.Index(l.trail, dep):])
		}
	}
}

// cycleText writes a cycle as "01 -> 02 -> 01".
func cycleText(cycle []*Order) string {
	nns := make([]string, 0, len(cycle)+1)
	for _, o := range cycle {
		nns = append(nns, o.NN)
	}
	return strings.Join(append(nns, cycle[0].NN), " -> ")
}

// check lists what is wrong with the plan's frontmatter.
func (pl *Plan) check(op string) (msgs []string) {
	if pl.Op != op {
		msgs = append(msgs, fmt.Sprintf("op %q, directory is %q", pl.Op, op))
	}
	if !slices.Contains(planStatuses, pl.Status) {
		msgs = append(msgs, fmt.Sprintf("unknown status %q", pl.Status))
	}
	return msgs
}

// check lists what is wrong with the order's frontmatter.
func (o *Order) check(op string) (msgs []string) {
	if want := fmt.Sprintf("%s-%s.md", o.NN, o.Slug); path.Base(o.Path) != want {
		msgs = append(msgs, "frontmatter says "+want)
	}
	if o.Op != op {
		msgs = append(msgs, fmt.Sprintf("op %q, directory is %q", o.Op, op))
	}
	if !slices.Contains(ranks, o.Rank) {
		msgs = append(msgs, fmt.Sprintf("unknown rank %q", o.Rank))
	}
	if o.Risk != "" && !slices.Contains(risks, o.Risk) {
		msgs = append(msgs, fmt.Sprintf("unknown risk %q", o.Risk))
	}
	return msgs
}

// check lists what is wrong with the report's frontmatter.
func (r *Report) check() (msgs []string) {
	if want := fmt.Sprintf("%02d-%s.%d.md", r.Order, r.Slug, r.Attempt); path.Base(r.Path) != want {
		msgs = append(msgs, "frontmatter says "+want)
	}
	if !slices.Contains(reportStatuses, r.Status) {
		msgs = append(msgs, fmt.Sprintf("unknown status %q", r.Status))
	}
	return msgs
}

// settle orders the reports by attempt and takes status and staleness from
// the latest. It lists earlier attempts still marked in-progress: a claim
// that was never finished.
func (o *Order) settle() (msgs []string) {
	o.Status = "pending"
	if len(o.Reports) == 0 {
		return nil
	}
	slices.SortFunc(o.Reports, func(a, b *Report) int { return cmp.Compare(a.Attempt, b.Attempt) })
	last := o.Reports[len(o.Reports)-1]
	o.Status, o.Stale = last.Status, last.OrderRev < o.Rev
	for _, r := range o.Reports[:len(o.Reports)-1] {
		if r.Status == "in-progress" {
			msgs = append(msgs, fmt.Sprintf("attempt %d in-progress, but attempt %d exists", r.Attempt, last.Attempt))
		}
	}
	return msgs
}

func mdFiles(fsys fs.FS, dir string) []string {
	entries, _ := fs.ReadDir(fsys, dir)
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			out = append(out, path.Join(dir, e.Name()))
		}
	}
	return out
}

var errNoFrontmatter = errors.New("no frontmatter")

// decode fills v from the file's frontmatter. Unknown keys are reported but
// the known ones are still decoded.
func decode(fsys fs.FS, p string, v any) error {
	src, err := fs.ReadFile(fsys, p)
	if err != nil {
		return errNoFrontmatter
	}
	fm, _, ok := split(src)
	if !ok {
		return errNoFrontmatter
	}
	if err := yaml.UnmarshalStrict(fm, v); err != nil {
		yaml.Unmarshal(fm, v)
		msg := strings.TrimPrefix(err.Error(), "yaml: unmarshal errors:")
		return errors.New(strings.Join(strings.Fields(msg), " "))
	}
	return nil
}
