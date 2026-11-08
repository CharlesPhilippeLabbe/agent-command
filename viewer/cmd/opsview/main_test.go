package main

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/CharlesPhilippeLabbe/agent-command/viewer/ops"
)

func testServer(t *testing.T, dir string) *server {
	t.Helper()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { root.Close() })
	s := &server{root: root, fsys: root.FS(), dir: dir, name: dir, pats: []string{"**/*.md"}, subs: map[chan struct{}]bool{}}
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
	s := testServer(t, dir)
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
