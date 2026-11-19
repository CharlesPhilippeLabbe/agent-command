# Note format

Path: `.saved/ops/<op>/notes/<target file name without .md>.<hex>.md`. `<hex>`: 4 lowercase hex chars. Examples: `plan.3fa9.md`, `03-api.3fa9.md`, `03-api.2.3fa9.md` (note on report attempt 2). Style: [style.md](style.md).

Feedback on one block of a plan, order or report: from the commander, written in opsview, or from the critic. Waits in `notes/` for the General's next design round or debrief.

Writers:
- opsview: creates notes, edits NOTE, sets `status: resolved`, deletes notes.
- Critic: creates notes with `by: critic`. Never edits, resolves or deletes a note.
- General: sets `status: resolved` and writes RESOLUTION, only after commander greenlight.
- Officers never read or write notes.

```markdown
---
op: <op>
target: plan.md|orders/NN-<slug>.md|reports/NN-<slug>.K.md
line: <N>
status: open|resolved
created: YYYY-MM-DD HH:MM
by: critic
---
# QUOTE
~~~
<exact source lines of the noted block>
~~~

# NOTE
<commander text, verbatim; or critic finding>

# RESOLUTION
- <target>: <change>
```

- `target`: path relative to the op dir.
- `line`: 1-based line in target, frontmatter included, where the block started when written.
- `status`: `open` until answered, then `resolved`.
- `created`: when opsview or the critic wrote the note.
- `by`: `critic` for critic notes. Omitted for commander notes.
- QUOTE fence: backticks, one more than the longest backtick run in the quote, at least 3. (`~~~` above only to nest inside this example.)
- RESOLUTION: one bullet per change. Rejected note: `- no change: <why>`. Omitted while open.

## Rules

- The quote is the anchor. `line` is only a hint.
- Quote found at another line: the note follows it.
- Quote not found: the note is detached. It stays open.
- A note is never a greenlight.
