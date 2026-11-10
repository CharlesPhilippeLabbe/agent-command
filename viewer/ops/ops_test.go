package ops

import (
	"encoding/json"
	"io/fs"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

func file(s string) *fstest.MapFile { return &fstest.MapFile{Data: []byte(s)} }

const dir = "proj/.saved/ops/vacuum-robot"

func fixture() fstest.MapFS {
	return fstest.MapFS{
		dir + "/plan.md":              file("---\nop: vacuum-robot\nstatus: active\ncreated: 2026-10-20\n---\n# GOAL\nPLAN BODY\n"),
		dir + "/orders/01-base.md":    file("---\nop: vacuum-robot\norder: 01\nslug: base\nrank: soldier\nrev: 1\n---\nORDER BODY\n"),
		dir + "/orders/02-api.md":     file("---\nop: vacuum-robot\norder: 02\nslug: api\nrank: captain\nrisk: medium\nrev: 2\ndepends: [01]\n---\nORDER BODY\n\n# RISKS\n- RISK BODY\n"),
		dir + "/orders/08-ui.md":      file("---\nop: vacuum-robot\norder: 08\nslug: ui\nrank: major\nrev: 1\ndepends: [01, 02]\n---\nORDER BODY\n"),
		dir + "/reports/01-base.1.md": file("---\nop: vacuum-robot\norder: 01\nslug: base\nattempt: 1\norder_rev: 1\norder_sha: aaa\nrank: soldier\nstatus: blocked\ngreenlit: n/a\ndate: 2026-10-21\n---\nREPORT BODY\n"),
		dir + "/reports/01-base.2.md": file("---\nop: vacuum-robot\norder: 01\nslug: base\nattempt: 2\norder_rev: 1\norder_sha: bbb\nrank: soldier\nstatus: done\ngreenlit: yes\ndate: 2026-10-22\n---\nREPORT BODY\n"),
		dir + "/reports/02-api.1.md":  file("---\nop: vacuum-robot\norder: 02\nslug: api\nattempt: 1\norder_rev: 1\norder_sha: ccc\nrank: captain\nstatus: partial\ngreenlit: yes\ndate: 2026-10-23\n---\nREPORT BODY\n"),
		"proj/README.md":              file("# not an op\n"),
	}
}

// files lists every file of fsys, as the server's walk would with a
// pattern matching everything.
func files(fsys fs.FS) []string {
	var out []string
	Walk(fsys, func(string) bool { return true }, func(p string) { out = append(out, p) })
	return out
}

func TestDiscover(t *testing.T) {
	fsys := fixture()
	got := Discover(fsys, files(fsys))
	if len(got) != 1 {
		t.Fatalf("got %d operations, want 1", len(got))
	}
	op := got[0]
	if op.Project != "proj" || op.Name != "vacuum-robot" || op.Plan.Status != "active" || op.Plan.Created != "2026-10-20" {
		t.Errorf("operation = %+v, plan = %+v", op, op.Plan)
	}
	if len(op.Problems) != 0 {
		t.Errorf("problems: %v", op.Problems)
	}

	want := []struct {
		order   Num
		status  string
		waiting []Num
		stale   bool
	}{
		{1, "done", nil, false},
		{2, "partial", []Num{1}[:0], true}, // 01 done; rev 2 > report's order_rev 1
		{8, "pending", []Num{2}, false},
	}
	if len(op.Orders) != len(want) {
		t.Fatalf("got %d orders", len(op.Orders))
	}
	for i, w := range want {
		o := op.Orders[i]
		if o.Order != w.order || o.Status != w.status || o.Stale != w.stale {
			t.Errorf("order %d = %+v", i, o)
		}
		if len(o.Waiting) != len(w.waiting) || (len(w.waiting) > 0 && !reflect.DeepEqual(o.Waiting, w.waiting)) {
			t.Errorf("order %s waiting %v, want %v", o.NN, o.Waiting, w.waiting)
		}
	}
	for i, col := range []int{1, 2, 3} {
		if op.Orders[i].Column != col {
			t.Errorf("order %s column %d, want %d", op.Orders[i].NN, op.Orders[i].Column, col)
		}
	}
	if op.Columns != 3 || op.Done != 1 || op.Total != 3 {
		t.Errorf("columns %d done %d total %d", op.Columns, op.Done, op.Total)
	}
	if op.Orders[2].NN != "08" || !reflect.DeepEqual(op.Orders[2].Depends, []Num{1, 2}) {
		t.Errorf("order 08 = %+v", op.Orders[2])
	}
	if o := op.Orders[1]; o.Risk != "medium" {
		t.Errorf("order 02 risk %q", o.Risk)
	}
	if r := op.Orders[0].Reports; len(r) != 2 || r[1].Attempt != 2 || r[1].Date != "2026-10-22" || r[1].OrderSha != "bbb" {
		t.Errorf("reports = %+v", r)
	}

	js, _ := json.Marshal(got)
	if strings.Contains(string(js), "BODY") {
		t.Errorf("JSON contains markdown body: %s", js)
	}
}

func TestProblems(t *testing.T) {
	fsys := fixture()
	fsys[dir+"/orders/03-loop.md"] = file("---\nop: other-op\norder: 03\nslug: loopy\nrank: general\nrev: 1\ndepends: [04, 10]\ncolour: red\n---\n")
	fsys[dir+"/orders/04-loop.md"] = file("---\nop: vacuum-robot\norder: 04\nslug: loop\nrank: soldier\nrev: 1\ndepends: [03]\n---\n")
	fsys[dir+"/orders/05-dup.md"] = file("---\nop: vacuum-robot\norder: 04\nslug: dup\nrank: soldier\nrev: 1\n---\n")
	fsys[dir+"/orders/06-none.md"] = file("# no frontmatter\n")
	fsys[dir+"/reports/02-api.3.md"] = file("---\nop: vacuum-robot\norder: 02\nslug: api\nattempt: 2\nstatus: weird\n---\n")
	fsys[dir+"/reports/07-ghost.1.md"] = file("---\nop: vacuum-robot\norder: 07\nslug: ghost\nattempt: 1\nstatus: done\n---\n")

	fsys[dir+"/orders/09-risky.md"] = file("---\nop: vacuum-robot\norder: 09\nslug: risky\nrank: soldier\nrisk: extreme\nrev: 1\n---\n")
	fsys[dir+"/reports/09-risky.1.md"] = file("---\nop: vacuum-robot\norder: 09\nslug: risky\nattempt: 1\nstatus: in-progress\n---\n")
	fsys[dir+"/reports/09-risky.2.md"] = file("---\nop: vacuum-robot\norder: 09\nslug: risky\nattempt: 2\nstatus: refused\n---\n")

	op := Load(fsys, dir)
	all := strings.Join(op.Problems, "\n")
	for _, want := range []string{
		"03-loop.md: frontmatter says 03-loopy.md",
		`03-loop.md: op "other-op"`,
		`03-loop.md: unknown rank "general"`,
		"03-loop.md: line 7: field colour not found",
		"05-dup.md: order 04 already used by 04-loop.md",
		"06-none.md: no frontmatter",
		"02-api.3.md: frontmatter says 02-api.2.md",
		`02-api.3.md: unknown status "weird"`,
		"07-ghost.1.md: no order 07",
		"03-loop.md: depends on missing order 10",
		"dependency cycle: 03 -> 04 -> 03",
		`09-risky.md: unknown risk "extreme"`,
		"09-risky.md: attempt 1 in-progress, but attempt 2 exists",
	} {
		if !strings.Contains(all, want) {
			t.Errorf("missing problem %q in:\n%s", want, all)
		}
	}
	if o := op.Orders[2]; o.Order != 3 || o.Rev != 1 {
		t.Errorf("lenient decode after strict failure: %+v", o)
	}
}

func TestInProgress(t *testing.T) {
	fsys := fixture()
	fsys[dir+"/reports/02-api.2.md"] = file("---\nop: vacuum-robot\norder: 02\nslug: api\nattempt: 2\norder_rev: 2\norder_sha: ddd\nrank: captain\nstatus: in-progress\ngreenlit: yes\ndate: 2026-10-24\n---\n")
	op := Load(fsys, dir)
	if len(op.Problems) != 0 {
		t.Errorf("problems: %v", op.Problems)
	}
	if o := op.Orders[1]; o.Status != "in-progress" || o.Stale {
		t.Errorf("order 02 = %+v", o)
	}
	if w := op.Orders[2].Waiting; !reflect.DeepEqual(w, []Num{2}) {
		t.Errorf("order 08 waiting %v, want [2]", w)
	}
}

func TestDropped(t *testing.T) {
	fsys := fixture()
	fsys[dir+"/reports/08-ui.1.md"] = file("---\nop: vacuum-robot\norder: 08\nslug: ui\nattempt: 1\norder_rev: 1\norder_sha: eee\nrank: general\nstatus: dropped\ngreenlit: yes\ndate: 2026-10-24\n---\n")
	fsys[dir+"/orders/09-tail.md"] = file("---\nop: vacuum-robot\norder: 09\nslug: tail\nrank: soldier\nrev: 1\ndepends: [08]\n---\n")
	op := Load(fsys, dir)
	if len(op.Problems) != 0 {
		t.Errorf("problems: %v", op.Problems)
	}
	if o := op.Orders[2]; o.Status != "dropped" || len(o.Waiting) != 0 {
		t.Errorf("order 08 = %+v", o)
	}
	if o := op.Orders[3]; o.Status != "pending" || !reflect.DeepEqual(o.Waiting, []Num{8}) {
		t.Errorf("order 09 = %+v", o)
	}
	if op.Dropped != 1 || op.Done != 1 || op.Total != 4 {
		t.Errorf("dropped %d done %d total %d", op.Dropped, op.Done, op.Total)
	}
}

func TestSplitFrontmatter(t *testing.T) {
	meta, body, err := SplitFrontmatter([]byte("---\nop: x\ndepends: [01, 02]\n---\n# Title\n"))
	if err != nil || len(meta) != 2 || meta[0].Key != "op" || string(body) != "# Title\n" {
		t.Errorf("meta %v body %q err %v", meta, body, err)
	}
	meta, body, _ = SplitFrontmatter([]byte("# no fm\n---\n"))
	if meta != nil || string(body) != "# no fm\n---\n" {
		t.Errorf("no frontmatter: meta %v body %q", meta, body)
	}
	_, body, _ = SplitFrontmatter([]byte("---\r\nop: x\r\n---\r\nbody\r\n"))
	if string(body) != "body\n" {
		t.Errorf("crlf body %q", body)
	}
}

// Discover loads only the plans in its list: a plan the pattern doesn't
// match makes no operation.
func TestDiscoverOnlyListed(t *testing.T) {
	fsys := fixture()
	var list []string
	for _, p := range files(fsys) {
		if p != dir+"/plan.md" {
			list = append(list, p)
		}
	}
	if got := Discover(fsys, list); len(got) != 0 {
		t.Fatalf("got %d operations, want 0", len(got))
	}
}
