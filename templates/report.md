# Report format

Path: `.saved/ops/<op>/reports/NN-<slug>.K.md`. K = attempt, starting at 1. Create `reports/` if missing. Never overwrite an earlier attempt, except your own `in-progress` attempt, which the final report replaces. Style: [style.md](style.md).

Always write a report, whatever the outcome. Only exception: order already `in-progress` in another session ([officer.md](officer.md) step 2).

Status:
- `in-progress`: greenlit, executing. Written at go, frontmatter only; claims the order. Never a final status.
- `done`: every STEP executed, every DONE WHEN check passes.
- `partial`: changes made, then stopped, or a DONE WHEN check fails.
- `blocked`: stopped before changing anything: ambiguity or hard blocker.
- `refused`: not taken: wrong rank, dependency not `done`, no new rev, order edited without rev bump, no greenlight.
- `dropped`: commander retired the order. General writes it at debrief, never an officer. Latest attempt `dropped` makes the order `dropped`. Allowed from any order not `done`, `partial` included.

```markdown
---
op: <op>
order: NN
slug: <slug>
attempt: K
order_rev: N
order_sha: <first 12 hex of sha256 of order file>
rank: <executing officer: soldier|captain|major; general for dropped>
status: in-progress|done|partial|blocked|refused|dropped
greenlit: yes|no|n/a
date: YYYY-MM-DD
---
# DONE
- file:line: change

# VERIFY
- `<command>`: verbatim output, trimmed

# ASSUMPTIONS
- <gray zone>: chose <X>. <why>.

# BLOCKERS
- <what stopped>. Q: <exact question for commander>

# DEVIATIONS
- Step N: <what differed>. <why>.

# SUGGESTIONS
- <useful extra not done, out of scope>
```

`order_sha`: `sha256sum <order> | cut -c1-12`.

`greenlit`: `yes` go given, `no` commander declined, `n/a` stopped before asking.

After writing: one line to commander with report path and status.

Stale `in-progress` (session died mid-run): commander deletes the attempt if nothing changed, else sets `status: partial` and adds BLOCKERS with the stop point. Then retry as usual.

## Dropped report

- General writes it, after commander greenlight, as next free attempt K.
- Never written while the latest attempt is `in-progress`. Commander resolves that claim first, per the stale rule above.
- Frontmatter: `rank: general`, `greenlit: yes`, `order_rev` and `order_sha` of the order file at drop time.
- Body has only REASON and LEFT BEHIND. Every other section omitted.
- REASON: why commander dropped the order. One fact per bullet.
- LEFT BEHIND: changes still in the repo, one `file:line: change` per bullet. Only when dropped from `partial`. Omit otherwise.

```markdown
# REASON
- <why commander dropped the order>

# LEFT BEHIND
- file:line: change
```
