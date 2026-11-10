package main

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/CharlesPhilippeLabbe/agent-command/viewer/ops"
)

func testServer(t *testing.T, dir string, pats ...string) *server {
	t.Helper()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { root.Close() })
	s := &server{root: root, fsys: root.FS(), dir: dir, name: dir, pats: pats, subs: map[chan struct{}]bool{}}
	if err := s.startWatch(); err != nil {
		t.Fatal(err)
	}
	return s
}

func write(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("# x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// waitFiles waits for the snapshot's files to equal want, draining pings.
func waitFiles(t *testing.T, s *server, ping chan struct{}, want []string) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for !slices.Equal(s.files(), want) {
		select {
		case <-ping:
		case <-deadline:
			t.Fatalf("files = %v, want %v", s.files(), want)
		}
	}
}

func TestWatchRescans(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "a.md"))
	s := testServer(t, dir, "**/*.md")
	ping := s.subscribe()
	if want := []string{"a.md"}; !slices.Equal(s.files(), want) {
		t.Fatalf("initial files = %v, want %v", s.files(), want)
	}

	write(t, filepath.Join(dir, "b.md"))
	waitFiles(t, s, ping, []string{"a.md", "b.md"})

	// a directory created after start is watched too
	write(t, filepath.Join(dir, "sub", "c.md"))
	waitFiles(t, s, ping, []string{"a.md", "b.md", "sub/c.md"})
	write(t, filepath.Join(dir, "sub", "d.md"))
	waitFiles(t, s, ping, []string{"a.md", "b.md", "sub/c.md", "sub/d.md"})

	if err := os.Remove(filepath.Join(dir, "a.md")); err != nil {
		t.Fatal(err)
	}
	waitFiles(t, s, ping, []string{"b.md", "sub/c.md", "sub/d.md"})
}

// visibleOps filters into a new slice; the snapshot's is shared by requests.
func TestVisibleOpsLeavesSnapshot(t *testing.T) {
	open := &ops.Operation{Name: "open", Plan: ops.Plan{Status: "active"}}
	closed := &ops.Operation{Name: "closed", Plan: ops.Plan{Status: "closed"}}
	s := &server{}
	s.snap.Store(&snapshot{ops: []*ops.Operation{closed, open}})

	got := s.visibleOps(httptest.NewRequest("GET", "/api/ops", nil))
	if len(got) != 1 || got[0] != open {
		t.Fatalf("visibleOps = %v, want only open", got)
	}
	if all := s.snap.Load().ops; len(all) != 2 || all[0] != closed || all[1] != open {
		t.Fatalf("snapshot changed: %v", all)
	}
}

func TestRelPattern(t *testing.T) {
	cases := []struct {
		name, pat, srcdir, want string
		fail                    bool
	}{
		{"relative unchanged", "**/.saved/ops/**/*.md", "/a/b", "**/.saved/ops/**/*.md", false},
		{"absolute under srcdir", "/a/b/*/.saved/ops/**/*.md", "/a/b", "*/.saved/ops/**/*.md", false},
		{"srcdir trailing slash", "/a/b/*/.saved/ops/**/*.md", "/a/b/", "*/.saved/ops/**/*.md", false},
		{"sibling sharing prefix", "/a/bc/x.md", "/a/b", "", true},
		{"absolute outside srcdir", "/tmp/*.md", "/a/b", "", true},
		{"srcdir root", "/home/x/*.md", "/", "home/x/*.md", false},
		{"pattern equal to srcdir", "/a/b", "/a/b", "", true},
		{"pattern climbing out", "/a/b/../c/x.md", "/a/b", "", true},
		{"dir named ..x", "/a/b/..x/y.md", "/a/b", "..x/y.md", false},
	}
	for _, c := range cases {
		got, err := relPattern(c.pat, c.srcdir)
		if c.fail {
			if err == nil {
				t.Errorf("%s: relPattern(%q, %q) = %q, want error", c.name, c.pat, c.srcdir, got)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("%s: relPattern(%q, %q) = %q, %v; want %q", c.name, c.pat, c.srcdir, got, err, c.want)
		}
	}
}

func TestReachable(t *testing.T) {
	const (
		anyDepth = "**/.saved/ops/**/*.md"
		oneLevel = "*/.saved/ops/**/*.md"
		literal  = "proj/.saved/ops/x/plan.md"
		brace    = "{a/b,c}/*.md"
	)
	cases := []struct {
		pat, dir string
		want     bool
	}{
		{anyDepth, "proj/src", true},
		{oneLevel, "proj", true},
		{oneLevel, "proj/src", false},
		{oneLevel, "proj/.saved", true},
		{oneLevel, "proj/.saved/ops/x/orders", true},
		{literal, "proj/.saved/ops/x", true},
		{literal, "proj/.saved/ops/x/orders", false},
		{literal, "other", false},
		{brace, "a/b", true},
		{anyDepth, ".", true},
		{oneLevel, ".", true},
		{literal, ".", true},
		{brace, ".", true},
	}
	for _, c := range cases {
		t.Run(c.pat+" "+c.dir, func(t *testing.T) {
			s := &server{pats: []string{c.pat}}
			if got := s.reachable(c.dir); got != c.want {
				t.Errorf("reachable(%q) with %q = %v, want %v", c.dir, c.pat, got, c.want)
			}
		})
	}
}

// The watcher skips directories the pattern can't reach.
func TestWatcherPrunes(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "proj", ".saved", "ops", "x", "plan.md"))
	write(t, filepath.Join(dir, "proj", "src", "a.md"))
	s := &server{pats: []string{"*/.saved/ops/**/*.md"}}
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { fw.Close() })
	w := &treeWatcher{Watcher: fw, root: dir, enter: s.reachable, dirs: map[string]bool{}}
	w.add(dir)
	if !w.dirs[filepath.Join(dir, "proj", ".saved", "ops")] {
		t.Errorf("proj/.saved/ops not watched: %v", w.dirs)
	}
	if w.dirs[filepath.Join(dir, "proj", "src")] {
		t.Errorf("proj/src watched: %v", w.dirs)
	}
}

// Only operations whose plan the pattern matches make the snapshot.
func TestSnapshotFollowsPattern(t *testing.T) {
	dir := t.TempDir()
	for _, p := range []string{"a", "b"} {
		full := filepath.Join(dir, p, ".saved", "ops", "op-"+p, "plan.md")
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("---\nop: op-"+p+"\nstatus: active\ncreated: 2026-11-09\n---\n# GOAL\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s := testServer(t, dir, "a/.saved/ops/**/*.md")
	got := s.snap.Load().ops
	if len(got) != 1 || got[0].Name != "op-a" {
		t.Fatalf("ops = %v, want only op-a", got)
	}
}

// noteOp is a tree with one operation whose notes/ directory already exists,
// so adding a note isn't a directory change.
func noteOp(t *testing.T) (dir, plan, notes string) {
	t.Helper()
	dir = t.TempDir()
	plan = filepath.Join(dir, "proj", ".saved", "ops", "x", "plan.md")
	notes = filepath.Join(dir, "proj", ".saved", "ops", "x", "notes")
	if err := os.MkdirAll(notes, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(plan, []byte("---\nop: x\nstatus: active\ncreated: 2026-11-09\n---\n# GOAL\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir, plan, notes
}

// A note-only change patches the file list and keeps the operations: no
// rescan ran.
func TestWatchNotesPatch(t *testing.T) {
	dir, _, notes := noteOp(t)
	s := testServer(t, dir, "**/*.md")
	ping := s.subscribe()
	op := s.snap.Load().ops[0]
	const planPath, notePath = "proj/.saved/ops/x/plan.md", "proj/.saved/ops/x/notes/plan.aaaa.md"

	write(t, filepath.Join(notes, "plan.aaaa.md"))
	waitFiles(t, s, ping, []string{notePath, planPath})
	if s.snap.Load().ops[0] != op {
		t.Errorf("operations reloaded on a note change")
	}

	if err := os.Remove(filepath.Join(notes, "plan.aaaa.md")); err != nil {
		t.Fatal(err)
	}
	waitFiles(t, s, ping, []string{planPath})
	if s.snap.Load().ops[0] != op {
		t.Errorf("operations reloaded on a note removal")
	}
}

// A note and another markdown change in one settle window rescan in full.
func TestWatchNotesWithPlanRescans(t *testing.T) {
	dir, plan, notes := noteOp(t)
	s := testServer(t, dir, "**/*.md")
	ping := s.subscribe()
	op := s.snap.Load().ops[0]

	if err := os.WriteFile(plan, []byte("---\nop: x\nstatus: issued\ncreated: 2026-11-09\n---\n# GOAL\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(notes, "plan.bbbb.md"))
	waitFiles(t, s, ping, []string{"proj/.saved/ops/x/notes/plan.bbbb.md", "proj/.saved/ops/x/plan.md"})
	if s.snap.Load().ops[0] == op {
		t.Errorf("operations kept after a plan change")
	}
}
