package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/CharlesPhilippeLabbe/agent-command/viewer/ops"
)

// The only writes opsview makes: commander notes, one file each in an
// operation's notes/ directory (templates/note.md). Plan, order and report
// files are never written.

// noteRequest is the JSON body of POST and PUT /api/notes. htmx sends form
// values as strings, so line and end come as decimal strings (the string
// option). Text and Status are pointers so PUT can tell which one was sent.
type noteRequest struct {
	Target string  `json:"target"`
	Line   int     `json:"line,string"`
	End    int     `json:"end,string"`
	Path   string  `json:"path"`
	Text   *string `json:"text"`
	Status *string `json:"status"`
}

// noteTries is how many random names a new note tries before giving up.
const noteTries = 16

// Creates a note on a block of a plan, order or report: lines line to end of
// the target file become its quote.
func (s *server) apiNoteCreate(w http.ResponseWriter, r *http.Request) {
	var req noteRequest
	if !readJSON(w, r, &req) {
		return
	}
	target := cleanPath(req.Target)
	op, rel, ok := s.noteTarget(w, target)
	if !ok {
		return
	}
	text := ""
	if req.Text != nil {
		text = strings.TrimSpace(*req.Text)
	}
	if text == "" {
		writeJSONError(w, http.StatusBadRequest, "empty note")
		return
	}
	src, err := fs.ReadFile(s.fsys, target)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, err.Error())
		return
	}
	quote, err := ops.Quote(src, req.Line, req.End)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	n := &ops.Note{
		Op: op.Name, Target: rel, Line: req.Line, Status: "open",
		Created: time.Now().Format("2006-01-02 15:04"),
		Quote:   quote, Text: text,
	}
	n.Path, err = s.createNote(path.Join(op.Dir, "notes"), rel, ops.FormatNote(n))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, n)
}

// createNote writes a new note file in dir under a random free name and
// returns its path.
func (s *server) createNote(dir, target string, data []byte) (string, error) {
	if err := s.root.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	for range noteTries {
		b := make([]byte, 2)
		rand.Read(b)
		p := path.Join(dir, ops.NoteName(target, hex.EncodeToString(b)))
		f, err := s.root.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		_, err = f.Write(data)
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		return p, err
	}
	return "", errors.New("no free note name")
}

// Edits an open note's text, or resolves it.
func (s *server) apiNoteUpdate(w http.ResponseWriter, r *http.Request) {
	var req noteRequest
	if !readJSON(w, r, &req) {
		return
	}
	p, ok := s.notePath(w, req.Path)
	if !ok {
		return
	}
	src, err := fs.ReadFile(s.fsys, p)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, err.Error())
		return
	}
	n, err := ops.ParseNote(src)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if n.Status != "open" {
		writeJSONError(w, http.StatusBadRequest, "note is resolved")
		return
	}
	switch {
	case (req.Text == nil) == (req.Status == nil):
		writeJSONError(w, http.StatusBadRequest, "want text or status")
		return
	case req.Text != nil:
		n.Text = strings.TrimSpace(*req.Text)
		if n.Text == "" {
			writeJSONError(w, http.StatusBadRequest, "empty note")
			return
		}
	case *req.Status != "resolved":
		writeJSONError(w, http.StatusBadRequest, "bad status: "+*req.Status)
		return
	default:
		n.Status = "resolved"
	}
	if err := s.root.WriteFile(p, ops.FormatNote(n), 0o644); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	n.Path = p
	writeJSON(w, http.StatusOK, n)
}

// Deletes a note, open or resolved.
func (s *server) apiNoteDelete(w http.ResponseWriter, r *http.Request) {
	p, ok := s.notePath(w, r.URL.Query().Get("path"))
	if !ok {
		return
	}
	if err := s.root.Remove(p); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": p})
}

// noteTarget finds the operation of a plan, order or report at served path
// p, and p relative to the operation's directory. A closed operation takes no
// new notes. On failure it writes the error and reports false.
func (s *server) noteTarget(w http.ResponseWriter, p string) (*ops.Operation, string, bool) {
	for _, op := range s.snap.Load().ops {
		if !isTargetOf(op, p) {
			continue
		}
		if op.Plan.Status == "closed" {
			writeJSONError(w, http.StatusBadRequest, "operation closed: "+op.Name)
			return nil, "", false
		}
		return op, strings.TrimPrefix(p, op.Dir+"/"), true
	}
	writeJSONError(w, http.StatusNotFound, "not a note target: "+p)
	return nil, "", false
}

// isTargetOf reports whether p is op's plan, one of its orders or one of
// their reports.
func isTargetOf(op *ops.Operation, p string) bool {
	if p == op.Plan.Path {
		return true
	}
	for _, o := range op.Orders {
		if p == o.Path || find(o.Reports, func(r *ops.Report) bool { return r.Path == p }) != nil {
			return true
		}
	}
	return false
}

// notePath checks that p is an existing note file: a .md file directly in
// an operation's notes/ directory, whatever the plan's status. On failure it
// writes the error and reports false.
func (s *server) notePath(w http.ResponseWriter, p string) (string, bool) {
	p = cleanPath(p)
	notes := path.Dir(p)
	if path.Ext(p) == ".md" && path.Base(notes) == "notes" && s.opAt(path.Dir(notes)) != nil {
		if info, err := s.root.Stat(p); err == nil && info.Mode().IsRegular() {
			return p, true
		}
	}
	writeJSONError(w, http.StatusNotFound, "no such note: "+p)
	return "", false
}

// readJSON decodes the request body into v. Only application/json is taken:
// browsers preflight it cross-site, so a form on another site can't post
// notes. A DNS-rebinding page counts as same-origin and gets past this.
func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mt != "application/json" {
		writeJSONError(w, http.StatusUnsupportedMediaType, "want application/json")
		return false
	}
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return false
	}
	return true
}
