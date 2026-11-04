package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/CharlesPhilippeLabbe/agent-command/viewer/ops"
)

type state string

const (
	stateSuccess state = "success"
	stateFailed  state = "failed"
)

// response is the envelope of every API answer.
type response struct {
	State state `json:"state"`
	Data  any   `json:"data,omitempty"`
	Error any   `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	writeResponse(w, status, &response{State: stateSuccess, Data: data})
}

// writeJSONError answers {"state":"failed","error":msg}.
func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeResponse(w, status, &response{State: stateFailed, Error: msg})
}

func writeResponse(w http.ResponseWriter, status int, r *response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(r)
}

// node is one entry of /api/tree. Dir is always present (no omitempty) so a
// Mustache {{#dir}} inside a file never falls through to its parent directory.
type node struct {
	Name     string  `json:"name"`
	Path     string  `json:"path"`
	Href     string  `json:"href,omitempty"`
	Dir      bool    `json:"dir"`
	Children []*node `json:"children,omitempty"`
}

type metaItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// document is the only response that carries a markdown body. List endpoints
// (/api/tree, /api/ops) return paths and frontmatter fields only. Prev and
// Next are the files either side of it in the sidebar.
type document struct {
	Path string     `json:"path"`
	Meta []metaItem `json:"meta,omitempty"`
	Body string     `json:"body"`
	Prev *step      `json:"prev"`
	Next *step      `json:"next"`
}

type listing struct {
	Root  string  `json:"root"`
	Files []*node `json:"files"`
}

type project struct {
	Name string `json:"name"`
}

// opSummary is the operation a panel belongs to, without its orders.
type opSummary struct {
	Project  string   `json:"project"`
	Name     string   `json:"name"`
	Plan     ops.Plan `json:"plan"`
	Done     int      `json:"done"`
	Total    int      `json:"total"`
	Problems []string `json:"problems"`
}

func summary(op *ops.Operation) opSummary {
	return opSummary{op.Project, op.Name, op.Plan, op.Done, op.Total, op.Problems}
}

// panel is what /api/order and /api/plan return: one file's markdown body,
// the operation it belongs to and the stops either side of it on the board.
// Path is the file whose body this is; OrderPath, Order and Report are set
// for an order, and Report too when a report tab is open.
type panel struct {
	Path      string      `json:"path"`
	OrderPath string      `json:"orderPath,omitempty"`
	Op        opSummary   `json:"op"`
	Order     *ops.Order  `json:"order,omitempty"`
	Report    *ops.Report `json:"report,omitempty"`
	Body      string      `json:"body"`
	Prev      *step       `json:"prev"`
	Here      *step       `json:"here"`
	Next      *step       `json:"next"`
}

// step is one stop of the panel's prev/next walk.
type step struct {
	Href  string `json:"href"`
	Label string `json:"label"`
}

func docHref(p string) string   { return "/?path=" + url.QueryEscape(p) }
func planHref(p string) string  { return "/ops.html?plan=" + url.QueryEscape(p) }
func orderHref(p string) string { return "/ops.html?order=" + url.QueryEscape(p) }
func reportHref(order, report string) string {
	return orderHref(order) + "&report=" + url.QueryEscape(report)
}

// cleanPath makes a requested path relative to the served root; ".." can't
// climb above it.
func cleanPath(p string) string { return strings.TrimPrefix(path.Clean("/"+p), "/") }

func (s *server) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/tree", s.apiTree)
	mux.HandleFunc("GET /api/doc", s.apiDoc)
	mux.HandleFunc("GET /api/ops", s.apiOps)
	mux.HandleFunc("GET /api/projects", s.apiProjects)
	mux.HandleFunc("GET /api/order", s.apiOrder)
	mux.HandleFunc("GET /api/plan", s.apiPlan)
	mux.HandleFunc("GET /api/events", s.events)
	mux.HandleFunc("GET /files/{path...}", s.rawFile)
}

func (s *server) apiTree(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"root": s.name, "nodes": s.tree()})
}

// One matched file with its frontmatter as rows, or without ?path= the list
// of every matched file.
func (s *server) apiDoc(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Query().Get("path")
	if p == "" {
		writeJSON(w, http.StatusOK, listing{Root: s.name, Files: s.fileNodes()})
		return
	}
	p = cleanPath(p)
	if !s.match(p) {
		writeJSONError(w, http.StatusNotFound, "not a matched markdown file: "+p)
		return
	}
	src, err := fs.ReadFile(s.fsys, p)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, err.Error())
		return
	}
	meta, body, err := ops.SplitFrontmatter(src)
	doc := document{Path: p, Meta: metaRows(meta, err), Body: string(body)}
	doc.Prev, _, doc.Next = neighbours(s.docSteps(), docHref(p))
	writeJSON(w, http.StatusOK, doc)
}

// docSteps lists the matched files as stops, in sidebar order.
func (s *server) docSteps() []step {
	var stops []step
	for _, p := range s.files() {
		stops = append(stops, step{docHref(p), p})
	}
	return stops
}

func (s *server) apiOps(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.visibleOps(r))
}

// Every project with at least one operation (closed included), for the
// dashboard's project filter.
func (s *server) apiProjects(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, projects(s.snap.Load().ops))
}

// One order in full, or with ?report=<path> one of its reports instead.
func (s *server) apiOrder(w http.ResponseWriter, r *http.Request) {
	p := cleanPath(r.URL.Query().Get("path"))
	if path.Base(path.Dir(p)) != "orders" {
		writeJSONError(w, http.StatusNotFound, "not an order: "+p)
		return
	}
	var o *ops.Order
	op := s.opAt(path.Dir(path.Dir(p)))
	if op != nil {
		o = find(op.Orders, func(o *ops.Order) bool { return o.Path == p })
	}
	if o == nil {
		writeJSONError(w, http.StatusNotFound, "no such order: "+p)
		return
	}
	res := &panel{Path: p, OrderPath: p, Op: summary(op), Order: o}
	href := orderHref(p)
	if rp := r.URL.Query().Get("report"); rp != "" {
		res.Report = find(o.Reports, func(x *ops.Report) bool { return x.Path == rp })
		if res.Report == nil {
			writeJSONError(w, http.StatusNotFound, "no such report: "+rp)
			return
		}
		res.Path, href = rp, reportHref(p, rp)
	}
	s.writePanel(w, r, res, href)
}

// One operation's plan in full.
func (s *server) apiPlan(w http.ResponseWriter, r *http.Request) {
	p := cleanPath(r.URL.Query().Get("path"))
	op := s.opAt(path.Dir(p))
	if path.Base(p) != "plan.md" || !s.match(p) || op == nil {
		writeJSONError(w, http.StatusNotFound, "not a plan: "+p)
		return
	}
	s.writePanel(w, r, &panel{Path: p, Op: summary(op)}, planHref(p))
}

// writePanel adds the body of res.Path and the stops around href, on the
// board as /api/ops shows it for the same ?all= and ?project=.
func (s *server) writePanel(w http.ResponseWriter, r *http.Request, res *panel, href string) {
	src, err := fs.ReadFile(s.fsys, res.Path)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, err.Error())
		return
	}
	_, body, _ := ops.SplitFrontmatter(src)
	res.Body = string(body)
	res.Prev, res.Here, res.Next = neighbours(walk(s.visibleOps(r)), href)
	writeJSON(w, http.StatusOK, res)
}

// Raw files referenced from docs (images, non-markdown links).
func (s *server) rawFile(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, s.fsys, cleanPath(r.PathValue("path")))
}

// visibleOps is what the dashboard shows: open operations, or all of them
// with ?all=1, of every project or only ?project=<name>.
func (s *server) visibleOps(r *http.Request) []*ops.Operation {
	all := r.URL.Query().Get("all") == "1"
	project := r.URL.Query().Get("project")
	// A new slice: the snapshot's is shared.
	out := []*ops.Operation{}
	for _, op := range s.snap.Load().ops {
		if (all || op.Plan.Status != "closed") && (project == "" || op.Project == project) {
			out = append(out, op)
		}
	}
	return out
}

// opAt is the operation in directory dir, or nil.
func (s *server) opAt(dir string) *ops.Operation {
	return find(s.snap.Load().ops, func(op *ops.Operation) bool { return op.Dir == dir })
}

// tree nests the matched files by directory, rebuilt per request so new files
// show up on reload.
func (s *server) tree() []*node {
	root := &node{Dir: true}
	dirs := map[string]*node{".": root}
	for _, p := range s.files() {
		dir := dirNode(dirs, path.Dir(p))
		dir.Children = append(dir.Children, fileNode(p))
	}
	return root.Children
}

// dirNode returns the node of directory p, adding it, and any parent
// missing too, to dirs and the tree.
func dirNode(dirs map[string]*node, p string) *node {
	if n, ok := dirs[p]; ok {
		return n
	}
	n := &node{Name: path.Base(p), Path: p, Dir: true}
	dirs[p] = n
	parent := dirNode(dirs, path.Dir(p))
	parent.Children = append(parent.Children, n)
	return n
}

func fileNode(p string) *node { return &node{Name: path.Base(p), Path: p, Href: docHref(p)} }

// fileNodes lists the matched files flat.
func (s *server) fileNodes() []*node {
	nodes := []*node{}
	for _, p := range s.files() {
		nodes = append(nodes, fileNode(p))
	}
	return nodes
}

// projects lists each project of list once, by name.
func projects(list []*ops.Operation) []project {
	names := []string{}
	for _, op := range list {
		names = append(names, op.Project)
	}
	slices.Sort(names)
	out := []project{}
	for _, name := range slices.Compact(names) {
		out = append(out, project{name})
	}
	return out
}

// walk lists every stop of the panel's prev/next walk in board order: each
// plan, then every order followed by its reports.
func walk(list []*ops.Operation) []step {
	var stops []step
	for _, op := range list {
		stops = append(stops, step{planHref(op.Plan.Path), op.Name + " · plan"})
		for _, o := range op.Orders {
			stops = append(stops, orderSteps(op.Name, o)...)
		}
	}
	return stops
}

// orderSteps is one order's part of the walk: the order, then its reports.
func orderSteps(opName string, o *ops.Order) []step {
	label := opName + " · " + o.NN + "-" + o.Slug
	stops := []step{{orderHref(o.Path), label}}
	for _, rep := range o.Reports {
		stops = append(stops, step{reportHref(o.Path, rep.Path), fmt.Sprintf("%s · report %d", label, rep.Attempt)})
	}
	return stops
}

// neighbours returns the stop at href and the stops either side of it, nil
// at the ends or when href isn't on the walk.
func neighbours(stops []step, href string) (prev, here, next *step) {
	i := slices.IndexFunc(stops, func(st step) bool { return st.Href == href })
	if i < 0 {
		return nil, nil, nil
	}
	if i > 0 {
		prev = &stops[i-1]
	}
	if i+1 < len(stops) {
		next = &stops[i+1]
	}
	return prev, &stops[i], next
}

// find returns the first element of list that ok accepts, or nil.
func find[T any](list []*T, ok func(*T) bool) *T {
	if i := slices.IndexFunc(list, ok); i >= 0 {
		return list[i]
	}
	return nil
}

// metaRows turns frontmatter into display rows, a decode error first.
func metaRows(meta yaml.MapSlice, err error) []metaItem {
	var rows []metaItem
	if err != nil {
		rows = append(rows, metaItem{"frontmatter error", err.Error()})
	}
	for _, item := range meta {
		rows = append(rows, metaItem{fmt.Sprint(item.Key), metaValue(item.Value)})
	}
	return rows
}

// metaValue flattens a frontmatter value for display: lists as "a, b".
func metaValue(v any) string {
	list, ok := v.([]any)
	if !ok {
		return fmt.Sprint(v)
	}
	parts := make([]string, len(list))
	for i, x := range list {
		parts[i] = fmt.Sprint(x)
	}
	return strings.Join(parts, ", ")
}
