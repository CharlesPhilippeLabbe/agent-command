// opsview serves a directory of markdown files: a JSON API plus static pages
// that render it in the browser (htmx + Mustache + marked).
//
//	opsview [--addr 127.0.0.1] [--port 8888] [--srcdir .] [--pattern "**/.saved/ops/**/*.md" ...]
//
// Patterns are matched against paths relative to --srcdir. An absolute
// pattern is allowed when it is under --srcdir. Patterns also decide which
// operations show (those whose plan.md they match) and which directories are
// walked and watched; a pattern starting with ** walks all of --srcdir.
package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/bmatcuk/doublestar/v4"
)

//go:embed static/*
var assets embed.FS

const defaultPattern = "**/.saved/ops/**/*.md"

type server struct {
	root *os.Root // writes (notes only) go through it, so they can't escape the directory
	fsys fs.FS
	dir  string // the same directory as fsys, as a path, for the watcher
	name string
	pats []string

	// snap is what the API reads; the watcher swaps in a new one when files change.
	snap atomic.Pointer[snapshot]

	mu   sync.Mutex
	subs map[chan struct{}]bool
}

func main() {
	addr, dir, pats := parseFlags()
	root, err := os.OpenRoot(dir)
	if err != nil {
		log.Fatal(err)
	}
	s := &server{root: root, fsys: root.FS(), dir: dir, name: dir, pats: pats, subs: map[chan struct{}]bool{}}
	if err := s.startWatch(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("serving %s (%s) on http://%s\n", dir, strings.Join(pats, ", "), addr)
	log.Fatal(http.ListenAndServe(addr, s.handler()))
}

func parseFlags() (addr, dir string, pats []string) {
	host := flag.String("addr", "127.0.0.1", "address to bind; 0.0.0.0 exposes the server to the network")
	port := flag.Int("port", 8888, "port to listen on")
	flag.StringVar(&dir, "srcdir", ".", "directory of markdown files to serve")
	var raw []string
	flag.Func("pattern", "glob of markdown files to show and watch, ** allowed, absolute only under srcdir; repeatable (default "+defaultPattern+")", func(v string) error {
		raw = append(raw, v)
		return nil
	})
	flag.Parse()
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}
	// Patterns are checked only now: an absolute one is made relative to
	// srcdir, which the positional argument may have set.
	for _, v := range raw {
		rel, err := relPattern(v, dir)
		if err != nil {
			log.Fatal(err)
		}
		v = normalizePattern(rel)
		if !doublestar.ValidatePattern(v) {
			log.Fatalf("bad pattern %q", v)
		}
		pats = append(pats, v)
	}
	if len(pats) == 0 {
		pats = []string{defaultPattern}
	}
	return net.JoinHostPort(*host, strconv.Itoa(*port)), dir, pats
}

// relPattern makes an absolute pattern relative to srcdir, since patterns are
// matched against paths below it. A relative pattern is returned as is; an
// absolute one outside srcdir, or srcdir itself, is an error. filepath.Rel
// compares whole segments, so srcdir /a/b doesn't take /a/bc/x.md, and it
// cleans the pattern lexically: a/./b becomes a/b. Symlinks are not resolved.
func relPattern(pat, srcdir string) (string, error) {
	if !filepath.IsAbs(pat) {
		return pat, nil
	}
	root, err := filepath.Abs(srcdir)
	if err != nil {
		return "", err
	}
	root = filepath.Clean(root)
	rel, err := filepath.Rel(root, pat)
	if err != nil {
		return "", err
	}
	rel = filepath.ToSlash(rel)
	// Only a whole ".." segment climbs out: a directory named "..x" is inside.
	if rel == "." || rel == ".." || strings.HasPrefix(rel, "../") {
		return "", fmt.Errorf("pattern %q is outside srcdir %q", pat, root)
	}
	return rel, nil
}

// handler serves the API and the embedded pages.
func (s *server) handler() http.Handler {
	// The system mime table may lack .js or give it no charset.
	mime.AddExtensionType(".js", "text/javascript; charset=utf-8")
	static, _ := fs.Sub(assets, "static")
	mux := http.NewServeMux()
	s.routes(mux)
	mux.Handle("/", http.FileServerFS(static))
	return mux
}

func (s *server) subscribe() chan struct{} {
	ch := make(chan struct{}, 1)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subs[ch] = true
	return ch
}

func (s *server) unsubscribe(ch chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.subs, ch)
}

// notify pings every subscriber, skipping any that already has a ping waiting.
func (s *server) notify() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// events is a server-sent event stream; each message tells the page to reload.
func (s *server) events(w http.ResponseWriter, r *http.Request) {
	ch := s.subscribe()
	defer s.unsubscribe(ch)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	rc := http.NewResponseController(w)
	fmt.Fprint(w, ": connected\n\n")
	rc.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ch:
			fmt.Fprint(w, "data: reload\n\n")
			rc.Flush()
		}
	}
}

// files lists the matched markdown files, in lexical order. The slice is
// shared: read it, don't change it.
func (s *server) files() []string { return s.snap.Load().files }

func (s *server) match(p string) bool {
	if !strings.HasSuffix(p, ".md") {
		return false
	}
	for _, pat := range s.pats {
		if ok, _ := doublestar.Match(pat, p); ok {
			return true
		}
	}
	return false
}

// reachable reports whether any pattern can match a file below dir, a slash
// path relative to srcdir. The walk and the watcher skip directories it
// rejects, so when unsure it says yes.
func (s *server) reachable(dir string) bool {
	if dir == "." {
		return true
	}
	segs := strings.Split(dir, "/")
	for _, pat := range s.pats {
		if patternReaches(pat, segs) {
			return true
		}
	}
	return false
}

// patternReaches compares a directory's segments, one by one, with the
// pattern's: a ** segment reaches anything below it, a mismatch nothing.
// The pattern's last segment names files, so a directory needs fewer
// segments than the pattern has.
func patternReaches(pat string, segs []string) bool {
	if braceHasSlash(pat) {
		return true // segments can't be compared one by one
	}
	psegs := strings.Split(pat, "/")
	for i, seg := range segs {
		if i < len(psegs) && psegs[i] == "**" {
			return true
		}
		if i >= len(psegs)-1 {
			return false
		}
		ok, err := doublestar.Match(psegs[i], seg)
		if err != nil {
			return true
		}
		if !ok {
			return false
		}
	}
	return true
}

// braceHasSlash reports whether pat has a / inside {}.
func braceHasSlash(pat string) bool {
	depth := 0
	for _, c := range pat {
		switch {
		case c == '{':
			depth++
		case c == '}' && depth > 0:
			depth--
		case c == '/' && depth > 0:
			return true
		}
	}
	return false
}

// normalizePattern rewrites a segment like "**.md" to "**/*.md", since doublestar
// only treats "**" as recursive when it is a whole path segment.
func normalizePattern(pat string) string {
	segs := strings.Split(pat, "/")
	for i, seg := range segs {
		if strings.HasPrefix(seg, "**") && seg != "**" {
			segs[i] = "**/*" + strings.TrimLeft(seg, "*")
		}
	}
	return strings.Join(segs, "/")
}
