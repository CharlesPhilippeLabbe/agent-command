package main

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
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

// rescan loads a fresh snapshot.
func (s *server) rescan() {
	snap := &snapshot{ops: ops.Discover(s.fsys)}
	ops.Walk(s.fsys, func(p string) {
		if s.match(p) {
			snap.files = append(snap.files, p)
		}
	})
	s.snap.Store(snap)
}

// startWatch loads the first snapshot and keeps it current in the background.
// The directories are watched before the first scan, so no change falls between.
func (s *server) startWatch() error {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	w := &treeWatcher{Watcher: fw, dirs: map[string]bool{}}
	w.add(s.dir)
	s.rescan()
	go s.watch(w)
	return nil
}

// treeWatcher is an fsnotify watcher over a whole tree. fsnotify watches
// directories, not trees, so every directory is added, and new ones as they
// appear.
type treeWatcher struct {
	*fsnotify.Watcher
	dirs   map[string]bool // the directories watched
	warned bool
}

// add watches root and every directory below it that isn't skipped.
func (w *treeWatcher) add(root string) {
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		switch {
		case err != nil || !d.IsDir():
		case p != root && ops.Skipped(d.Name()):
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

// settle is how long changes must stop coming before the tree is rescanned:
// an editor's temp-file-and-rename or a git checkout is a burst of events.
const settle = 100 * time.Millisecond

// watch rescans and pings the subscribers whenever a markdown file or a
// directory is added, removed or modified.
func (s *server) watch(w *treeWatcher) {
	var due <-chan time.Time
	for {
		select {
		case evt := <-w.Events:
			if evt.Has(fsnotify.Create) {
				if info, err := os.Stat(evt.Name); err == nil && info.IsDir() {
					w.add(evt.Name)
					due = time.After(settle)
				}
			}
			if w.dirs[evt.Name] && evt.Has(fsnotify.Remove|fsnotify.Rename) {
				delete(w.dirs, evt.Name)
				due = time.After(settle)
			}
			if strings.HasSuffix(evt.Name, ".md") && evt.Op != fsnotify.Chmod {
				due = time.After(settle)
			}
		case err := <-w.Errors:
			// Most likely an event queue overflow: events were lost, so
			// whatever the snapshot holds may be stale.
			log.Printf("watch: %v", err)
			due = time.After(settle)
		case <-due:
			due = nil
			s.rescan()
			s.notify()
		}
	}
}
