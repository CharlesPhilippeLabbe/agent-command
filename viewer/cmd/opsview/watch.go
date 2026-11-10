package main

import (
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/CharlesPhilippeLabbe/agent-command/viewer/ops"
)

// snapshot is everything the API reads from the tree, loaded once and
// replaced whole when a file changes, so requests never walk the disk.
type snapshot struct {
	files []string         // matched markdown, lexical
	ops   []*ops.Operation // every operation, closed included
}

// rescan loads a fresh snapshot in one walk, entering only directories the
// patterns can reach.
func (s *server) rescan() {
	snap := &snapshot{}
	ops.Walk(s.fsys, s.reachable, func(p string) {
		if s.match(p) {
			snap.files = append(snap.files, p)
		}
	})
	snap.ops = ops.Discover(s.fsys, snap.files)
	s.snap.Store(snap)
}

// startWatch loads the first snapshot and keeps it current in the background.
// The directories are watched before the first scan, so no change falls between.
func (s *server) startWatch() error {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	w := &treeWatcher{Watcher: fw, root: s.dir, enter: s.reachable, dirs: map[string]bool{}}
	w.add(s.dir)
	s.rescan()
	go s.watch(w)
	return nil
}

// treeWatcher is an fsnotify watcher over a whole tree. fsnotify watches
// directories, not trees, so every directory is added, and new ones as they
// appear. Only directories the patterns can reach are watched.
type treeWatcher struct {
	*fsnotify.Watcher
	root   string                // the served directory, as an OS path
	enter  func(dir string) bool // whether a directory, as a slash path below root, is watched
	dirs   map[string]bool       // the directories watched
	warned bool
}

// add watches top and every directory below it that isn't skipped. The
// served root itself is always watched.
func (w *treeWatcher) add(top string) {
	filepath.WalkDir(top, func(p string, d fs.DirEntry, err error) error {
		switch {
		case err != nil || !d.IsDir():
		case p != w.root && (ops.Skipped(d.Name()) || !w.enter(w.rel(p))):
			return filepath.SkipDir
		case !w.dirs[p]:
			if err := w.Add(p); err != nil {
				if !w.warned {
					log.Printf("watch %s: %v (more directories than fs.inotify.max_user_watches allows?)", p, err)
					w.warned = true
				}
				break
			}
			w.dirs[p] = true
		}
		return nil
	})
}

// rel is OS path p as a slash path below the served root, the form patterns
// are matched against.
func (w *treeWatcher) rel(p string) string {
	r, err := filepath.Rel(w.root, p)
	if err != nil {
		return p
	}
	return filepath.ToSlash(r)
}

// settle is how long changes must stop coming before the tree is rescanned:
// an editor's temp-file-and-rename or a git checkout is a burst of events.
const settle = 100 * time.Millisecond

// watch rescans and pings the subscribers whenever a markdown file or a
// directory is added, removed or modified. When only note files changed in
// a settle window, the file list is patched instead of walking the tree:
// notes aren't part of the operations, so nothing else can be stale.
func (s *server) watch(w *treeWatcher) {
	var due <-chan time.Time
	full := false
	notes := map[string]bool{} // note files changed, as slash paths below the root
	for {
		select {
		case evt := <-w.Events:
			if evt.Has(fsnotify.Create) {
				if info, err := os.Stat(evt.Name); err == nil && info.IsDir() {
					w.add(evt.Name)
					full = true
					due = time.After(settle)
				}
			}
			if w.dirs[evt.Name] && evt.Has(fsnotify.Remove|fsnotify.Rename) {
				delete(w.dirs, evt.Name)
				full = true
				due = time.After(settle)
			}
			if strings.HasSuffix(evt.Name, ".md") && evt.Op != fsnotify.Chmod {
				if p := w.rel(evt.Name); s.isNote(p) {
					notes[p] = true
				} else {
					full = true
				}
				due = time.After(settle)
			}
		case err := <-w.Errors:
			// Most likely an event queue overflow: events were lost, so
			// whatever the snapshot holds may be stale.
			log.Printf("watch: %v", err)
			full = true
			due = time.After(settle)
		case <-due:
			due = nil
			if full {
				s.rescan()
			} else {
				s.patchNotes(notes)
			}
			s.notify()
			full = false
			clear(notes)
		}
	}
}

// isNote reports whether p is a note file: a .md file directly in the notes/
// directory of an operation in the current snapshot.
func (s *server) isNote(p string) bool {
	notes := path.Dir(p)
	if path.Ext(p) != ".md" || path.Base(notes) != "notes" {
		return false
	}
	return s.opAt(path.Dir(notes)) != nil
}

// patchNotes stores a new snapshot whose file list has each changed note
// file in it if it exists and is matched, out of it otherwise. The
// operations are kept: the live snapshot is shared, so it is never changed.
func (s *server) patchNotes(changed map[string]bool) {
	old := s.snap.Load()
	set := map[string]bool{}
	for _, p := range old.files {
		set[p] = true
	}
	for p := range changed {
		info, err := fs.Stat(s.fsys, p)
		set[p] = err == nil && info.Mode().IsRegular() && s.match(p)
	}
	files := []string{}
	for p, in := range set {
		if in {
			files = append(files, p)
		}
	}
	slices.SortFunc(files, walkOrder)
	s.snap.Store(&snapshot{files: files, ops: old.ops})
}

// walkOrder compares two paths in the order a walk lists them: segment by
// segment, so "a/b.md" comes before "a.md" as it does in the walk, where a
// plain string sort puts it after.
func walkOrder(a, b string) int {
	return strings.Compare(strings.ReplaceAll(a, "/", "\x00"), strings.ReplaceAll(b, "/", "\x00"))
}
