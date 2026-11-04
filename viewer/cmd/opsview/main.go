// opsview serves a directory of markdown files: a JSON API plus static pages
// that render it in the browser (htmx + Mustache + marked).
//
//	opsview [--addr 127.0.0.1] [--port 8888] [--srcdir .] [--pattern "**/.saved/ops/**/*.md" ...]
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
	s := &server{fsys: root.FS(), dir: dir, name: dir, pats: pats, subs: map[chan struct{}]bool{}}
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
	flag.Func("pattern", "glob of markdown files to show and watch, ** allowed; repeatable (default "+defaultPattern+")", func(v string) error {
		v = normalizePattern(v)
		if !doublestar.ValidatePattern(v) {
			return fmt.Errorf("bad pattern %q", v)
		}
		pats = append(pats, v)
		return nil
	})
	flag.Parse()
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}
	if len(pats) == 0 {
		pats = []string{defaultPattern}
	}
	return net.JoinHostPort(*host, strconv.Itoa(*port)), dir, pats
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
