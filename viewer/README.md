# opsview

Local browser view of [ops](../README.md) operations: every `.saved/ops/<op>/` under a directory, as dependency boards with order, report and plan panels, plus a browser for the markdown files themselves.

The markdown files stay the only source of truth. opsview reads them on each request and writes only note files: commander notes in an operation's `notes/` ([note format](../skills/shared/note.md)).

## Build and run

Needs Go 1.27+.

```sh
go install github.com/CharlesPhilippeLabbe/agent-command/viewer/cmd/opsview@latest
opsview --srcdir ~/code          # every repo under ~/code with .saved/ops/
```

From a checkout: `cd viewer && go build -o opsview ./cmd/opsview`.

Open http://127.0.0.1:8888/ops.html for the dashboard, or http://127.0.0.1:8888/ for the docs browser.

| Flag | Default | |
|---|---|---|
| `--srcdir` | `.` | directory to scan; a positional argument does the same |
| `--port` | `8888` | |
| `--addr` | `127.0.0.1` | bind address |
| `--pattern` | `**/.saved/ops/**/*.md` | glob of files the docs browser shows and the watcher follows; `**` allowed, repeatable |

Directories named `.git`, `node_modules` and `.venv` are skipped.

### Development

With [air](https://github.com/air-verse/air), `air` in `viewer/` rebuilds on any Go, HTML, CSS or JS change and serves `../..` on the default port (see `.air.toml`). The open page reloads itself after a rebuild.

```sh
go vet ./... && go test ./...
```

## Pages

**ops** (`/ops.html`): one board per open operation. Orders sit in columns by dependency depth, joined by their dependency lines; hovering a card highlights its chain.

- Card: number, slug, status, rank, rev, report count, risk level. `waiting` means a dependency isn't `done`; `revised` means the order changed since its latest report. `in-progress` pulses. A `dropped` order shows a struck-through slug and is never `waiting`; the `N dropped` chip next to `done/total` shows only when above 0.
- Filters: project chips, and "include closed".
- Panel: click a card, or the plan, for its frontmatter, reports and rendered body. Report tabs switch to one attempt. Prev and next step through every plan, order and report on the board. Three widths (a bottom sheet on screens up to 1024px wide).
- Problems found in the files (bad frontmatter, wrong file names, missing dependencies, cycles, an `in-progress` attempt that isn't the latest) are listed above the board.
- Notes: the panel's "annotate" toggle (remembered per browser) turns a tap on any block of the plan, order or report (heading, paragraph, list item, table row, code block, quote) into a note form under it. A saved note shows as a card under its block, with edit, resolve, and delete behind a confirm dialog. A note whose quoted lines are gone from the file sits in a "Detached notes" tray at the top. A closed operation has no toggle, so it takes no new notes, but its open notes still show and keep their buttons. While a note form holds text, hot reload leaves the panel alone and catches up once the form closes. Resolved notes sit in a collapsed "Resolved" section at the bottom, each with its quote, the note, the General's resolution and a delete button.

**docs** (`/`): tree of every matched markdown file, with frontmatter as a table and the body rendered.

Everything lives in the URL (`/ops.html?order=<path>&report=<path>&project=<name>&all=1`), so links and the back button work. Any change to a matched file reloads what's on screen. A RISKS section is boxed wherever it appears. Light and dark themes follow the system until toggled.

## API

JSON only, wrapped as `{"state": "success", "data": ...}` or `{"state": "failed", "error": "..."}`. Only the doc, order and plan endpoints return markdown bodies.

| Endpoint | Returns |
|---|---|
| `GET /api/ops[?all=1][&project=<name>]` | operations with plan, orders, reports and derived state |
| `GET /api/projects` | project names |
| `GET /api/order?path=<order>[&report=<report>]` | one order or report: frontmatter, body, open and resolved notes, prev and next |
| `GET /api/plan?path=<plan>` | one plan: operation summary, body, open and resolved notes, prev and next |
| `GET /api/tree` | matched files nested by directory |
| `GET /api/doc[?path=<file>]` | one matched file, or the flat list without `path` |
| `GET /api/events` | server-sent event on every change to a matched file |
| `GET /files/<path>` | raw file, for images and links inside docs |
| `POST /api/notes` | new note on lines `line` to `end` of `target` (a plan, order or report of an operation that isn't closed) |
| `PUT /api/notes` | an open note at `path` with new `text`, or `status: "resolved"` |
| `DELETE /api/notes?path=<note>` | the note file removed |

Derived per order: `status` (latest attempt's, else `pending`), `waiting`, `stale`, `column`.

Order and plan panels also carry `bodyLine` (the file line the body starts at), `annotatable` (the operation isn't closed) and `notes`: each open note on the file, with `line` moved to where its quote is now, or `detached` when the quote is gone. `resolved` lists the resolved notes on the file, oldest first.

## Security

- No authentication. The default bind is `127.0.0.1`. `--addr 0.0.0.0` lets anyone on the network read every file under `--srcdir` through `/files/`.
- Paths are cleaned and opened through `os.OpenRoot`, so requests can't escape `--srcdir`, symlinks included.
- Writes go only to note files directly in an operation's `notes/`, through `os.OpenRoot`. Plan, order and report files are never written.
- `POST` and `PUT /api/notes` take only `Content-Type: application/json`, so browsers send a preflight and a form on another site can't post notes. A DNS-rebinding page counts as same-origin and gets past this.
- No auth on writes either: anyone who reaches the port can create, edit and delete notes.

## Layout

```
cmd/opsview/
  main.go        flags, file matching, watcher, server-sent events
  api.go         JSON endpoints
  notes.go       note endpoints: create, edit, resolve, delete
  static/        pages, templates and styles, embedded in the binary
    app.js       shared: Mustache rendering for htmx, markdown, links, hot reload, theme
    docs.js      docs page
    ops.js       dashboard: panel, filters, dependency lines
ops/             reads operations from frontmatter, derives status, dependencies, problems; parses and anchors notes
```

The pages are static HTML. htmx fetches JSON and renders it with Mustache templates kept in the page (`mustache-tmpl` extension). marked renders markdown in the browser. All three load from pinned CDN versions.
