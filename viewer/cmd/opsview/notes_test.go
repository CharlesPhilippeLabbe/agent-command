package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CharlesPhilippeLabbe/agent-command/viewer/ops"
)

const (
	openOp = "p/.saved/ops/open-op"
	shutOp = "p/.saved/ops/shut-op"
	// openPlan's body starts at line 6; "Goal line." is line 7.
	openPlan = "---\nop: open-op\nstatus: issued\ncreated: 2026-11-07\n---\n# GOAL\nGoal line.\n\n# LOG\n- one\n"
)

// noteFixture is a tree with one open operation (plan, order, report), one
// closed operation holding an open note, and a README outside any operation.
func noteFixture(t *testing.T) (*server, string) {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		openOp + "/plan.md":              openPlan,
		openOp + "/orders/01-base.md":    "---\nop: open-op\norder: 01\nslug: base\nrank: soldier\nrev: 1\n---\n# OBJECTIVE\nBase.\n",
		openOp + "/reports/01-base.1.md": "---\nop: open-op\norder: 01\nslug: base\nattempt: 1\norder_rev: 1\norder_sha: aaa\nrank: soldier\nstatus: done\ngreenlit: yes\ndate: 2026-11-07\n---\n# DONE\n- x\n",
		shutOp + "/plan.md":              "---\nop: shut-op\nstatus: closed\ncreated: 2026-11-07\n---\n# GOAL\nOld goal.\n",
		shutOp + "/notes/plan.aaaa.md":   string(ops.FormatNote(&ops.Note{Op: "shut-op", Target: "plan.md", Line: 6, Status: "open", Created: "2026-11-07 10:00", Quote: "Old goal.", Text: "Still open."})),
		"p/README.md":                    "# readme\n",
	}
	for p, src := range files {
		full := filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return testServer(t, dir, "**/*.md"), dir
}

// call sends one request to s and decodes the envelope's data into out, when
// out isn't nil.
func call(t *testing.T, s *server, method, target, contentType, body string, out any) int {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	s.handler().ServeHTTP(rec, req)
	if out != nil && rec.Code == http.StatusOK {
		env := struct{ Data json.RawMessage }{}
		if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(env.Data, out); err != nil {
			t.Fatal(err)
		}
	}
	return rec.Code
}

func jsonBody(v map[string]string) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// create posts a note and returns it.
func create(t *testing.T, s *server, target, line, end, text string) *ops.Note {
	t.Helper()
	n := &ops.Note{}
	body := jsonBody(map[string]string{"target": target, "line": line, "end": end, "text": text})
	if code := call(t, s, "POST", "/api/notes", "application/json", body, n); code != http.StatusOK {
		t.Fatalf("POST %s: %d", target, code)
	}
	return n
}

// readNote parses the note file at served path p.
func readNote(t *testing.T, dir, p string) *ops.Note {
	t.Helper()
	src, err := os.ReadFile(filepath.Join(dir, p))
	if err != nil {
		t.Fatal(err)
	}
	n, err := ops.ParseNote(src)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestNoteCreate(t *testing.T) {
	s, dir := noteFixture(t)

	n := create(t, s, openOp+"/plan.md", "7", "7", "  Too vague.  ")
	if !strings.HasPrefix(n.Path, openOp+"/notes/plan.") {
		t.Errorf("path = %s", n.Path)
	}
	got := readNote(t, dir, n.Path)
	if got.Quote != "Goal line." || got.Text != "Too vague." || got.Status != "open" || got.Target != "plan.md" || got.Line != 7 || got.Op != "open-op" {
		t.Errorf("note file = %+v", got)
	}

	if n := create(t, s, openOp+"/orders/01-base.md", "8", "9", "order note"); readNote(t, dir, n.Path).Quote != "# OBJECTIVE\nBase." {
		t.Errorf("order note quote wrong")
	}
	if n := create(t, s, openOp+"/reports/01-base.1.md", "13", "13", "report note"); !strings.HasPrefix(n.Path, openOp+"/notes/01-base.1.") {
		t.Errorf("report note path = %s", n.Path)
	}

	bad := []struct {
		name, contentType string
		body              map[string]string
		want              int
	}{
		{"closed op", "application/json", map[string]string{"target": shutOp + "/plan.md", "line": "6", "end": "6", "text": "x"}, 400},
		{"outside ops", "application/json", map[string]string{"target": "p/README.md", "line": "1", "end": "1", "text": "x"}, 404},
		{"empty text", "application/json", map[string]string{"target": openOp + "/plan.md", "line": "7", "end": "7", "text": "  "}, 400},
		{"bad line", "application/json", map[string]string{"target": openOp + "/plan.md", "line": "x", "end": "7", "text": "x"}, 400},
		{"lines past end", "application/json", map[string]string{"target": openOp + "/plan.md", "line": "7", "end": "99", "text": "x"}, 400},
		{"not JSON", "text/plain", map[string]string{"target": openOp + "/plan.md", "line": "7", "end": "7", "text": "x"}, 415},
	}
	for _, c := range bad {
		if code := call(t, s, "POST", "/api/notes", c.contentType, jsonBody(c.body), nil); code != c.want {
			t.Errorf("%s: %d, want %d", c.name, code, c.want)
		}
	}
}

func TestNoteUpdate(t *testing.T) {
	s, dir := noteFixture(t)
	n := create(t, s, openOp+"/plan.md", "7", "7", "first")

	put := func(body map[string]string) int {
		return call(t, s, "PUT", "/api/notes", "application/json", jsonBody(body), nil)
	}
	if code := put(map[string]string{"path": n.Path, "text": "second"}); code != 200 {
		t.Fatalf("PUT text: %d", code)
	}
	if got := readNote(t, dir, n.Path); got.Text != "second" || got.Quote != "Goal line." {
		t.Errorf("after PUT text: %+v", got)
	}

	for name, c := range map[string]struct {
		body map[string]string
		want int
	}{
		"neither":    {map[string]string{"path": n.Path}, 400},
		"both":       {map[string]string{"path": n.Path, "text": "x", "status": "resolved"}, 400},
		"bad status": {map[string]string{"path": n.Path, "status": "open"}, 400},
		"empty text": {map[string]string{"path": n.Path, "text": " "}, 400},
		"plan path":  {map[string]string{"path": openOp + "/plan.md", "text": "x"}, 404},
		"escape":     {map[string]string{"path": "../" + openOp + "/notes/x.md", "text": "x"}, 404},
	} {
		if code := put(c.body); code != c.want {
			t.Errorf("%s: %d, want %d", name, code, c.want)
		}
	}
	if src, _ := os.ReadFile(filepath.Join(dir, openOp, "plan.md")); string(src) != openPlan {
		t.Errorf("plan changed:\n%s", src)
	}

	if code := put(map[string]string{"path": n.Path, "status": "resolved"}); code != 200 {
		t.Fatalf("PUT resolved: %d", code)
	}
	if got := readNote(t, dir, n.Path); got.Status != "resolved" {
		t.Errorf("status = %s", got.Status)
	}
	if code := put(map[string]string{"path": n.Path, "status": "resolved"}); code != 400 {
		t.Errorf("second resolve: %d, want 400", code)
	}

	if code := put(map[string]string{"path": shutOp + "/notes/plan.aaaa.md", "text": "edited"}); code != 200 {
		t.Errorf("PUT on closed op: %d, want 200", code)
	}
}

func TestNoteDelete(t *testing.T) {
	s, dir := noteFixture(t)
	n := create(t, s, openOp+"/plan.md", "7", "7", "doomed")

	del := func(p string) int {
		return call(t, s, "DELETE", "/api/notes?path="+url.QueryEscape(p), "", "", nil)
	}
	if code := del(openOp + "/plan.md"); code != 404 {
		t.Errorf("DELETE plan: %d, want 404", code)
	}
	if _, err := os.Stat(filepath.Join(dir, openOp, "plan.md")); err != nil {
		t.Errorf("plan gone: %v", err)
	}
	if code := del(n.Path); code != 200 {
		t.Fatalf("DELETE: %d", code)
	}
	if _, err := os.Stat(filepath.Join(dir, n.Path)); !os.IsNotExist(err) {
		t.Errorf("note still there: %v", err)
	}
	if code := del(shutOp + "/notes/plan.aaaa.md"); code != 200 {
		t.Errorf("DELETE on closed op: %d, want 200", code)
	}
}

func TestPanelNotes(t *testing.T) {
	s, _ := noteFixture(t)
	open := create(t, s, openOp+"/plan.md", "7", "7", "open one")
	done := create(t, s, openOp+"/plan.md", "9", "9", "resolved one")
	if code := call(t, s, "PUT", "/api/notes", "application/json", jsonBody(map[string]string{"path": done.Path, "status": "resolved"}), nil); code != 200 {
		t.Fatalf("resolve: %d", code)
	}

	var res panel
	if code := call(t, s, "GET", "/api/plan?path="+url.QueryEscape(openOp+"/plan.md"), "", "", &res); code != 200 {
		t.Fatalf("GET plan: %d", code)
	}
	if !res.Annotatable || res.BodyLine != 6 {
		t.Errorf("annotatable %v bodyLine %d, want true 6", res.Annotatable, res.BodyLine)
	}
	if len(res.Notes) != 1 || res.Notes[0].Path != open.Path || res.Notes[0].Line != 7 || res.Notes[0].Detached {
		t.Errorf("notes = %+v", res.Notes)
	}
	if len(res.Resolved) != 1 || res.Resolved[0].Path != done.Path {
		t.Errorf("resolved = %+v", res.Resolved)
	}

	var shut panel
	if code := call(t, s, "GET", "/api/plan?all=1&path="+url.QueryEscape(shutOp+"/plan.md"), "", "", &shut); code != 200 {
		t.Fatalf("GET closed plan: %d", code)
	}
	if shut.Annotatable || len(shut.Notes) != 1 || shut.Notes[0].Text != "Still open." {
		t.Errorf("closed: annotatable %v notes %+v", shut.Annotatable, shut.Notes)
	}
}
