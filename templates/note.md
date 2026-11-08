# Note format

Path: `.saved/ops/<op>/notes/<target file name without .md>.<hex>.md`. `<hex>`: 4 lowercase hex chars. Examples: `plan.3fa9.md`, `03-api.3fa9.md`, `03-api.2.3fa9.md` (note on report attempt 2). Style: [style.md](style.md).

A note is commander feedback on one block of a plan, order or report, written in opsview. It waits in `notes/` until the next General debrief.

Writers:
- opsview: creates notes, edits NOTE, sets `status: resolved`, deletes notes.
- General: sets `status: resolved` and writes RESOLUTION, only after commander greenlight.
- Officers never read or write notes.

```markdown
---
op: <op>
target: plan.md|orders/NN-<slug>.md|reports/NN-<slug>.K.md
line: <N>
status: open|resolved
created: YYYY-MM-DD HH:MM
---
# QUOTE
~~~
<exact source lines of the noted block>
~~~

# NOTE
<commander text, verbatim>

# RESOLUTION
- <target>: <change>
```

- `target`: path relative to the op dir.
- `line`: 1-based line of the target file, frontmatter included, where the noted block started when the note was written.
- `status`: `open` until answered, then `resolved`.
- `created`: when opsview wrote the note.
- QUOTE fence: backticks, one more than the longest backtick run in the quote, at least 3. (Shown as `~~~` above only to nest inside this example.)
- RESOLUTION: one bullet per change. Rejected note: `- no change: <why>`. Omitted while open.

## Rules

- The quote is the anchor. `line` is only a hint.
- Quote found at another line: the note follows it.
- Quote not found: the note is detached. It stays open.
- A note is never a greenlight.
