# Report format

Path: `.saved/ops/<op>/reports/NN-<slug>.K.md`. K = attempt, from 1. Create `reports/` if missing. Style: [style.md](style.md).

Always write a report. Only exception: order `in-progress` in another session ([officer.md](officer.md) step 2). Never overwrite an earlier attempt. Exceptions: your own `in-progress` attempt, replaced by the final report, and a [re-block](#re-block), appended.

Status:
- `in-progress`: greenlit, executing. Written at go, frontmatter only, claims the order. Never final.
- `done`: every STEP executed, every DONE WHEN check passes.
- `partial`: changes made, then stopped, or a DONE WHEN check fails.
- `blocked`: stopped before changing anything.
- `refused`: not taken: wrong rank, dependency not `done`, order unchanged, dropped without new rev, no greenlight.
- `dropped`: commander retired the order, from any status but `done`. Only General writes it.

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
- `<command>`: check defect. <why the check is wrong>. Output: <verbatim, trimmed>

# ASSUMPTIONS
- <gray zone>: chose <X>. <why>.

# BLOCKERS
- <what stopped>. Q: <exact question for commander>

# DEVIATIONS
- Step N: <what differed>. <why>.

# SUGGESTIONS
- <useful extra not done, out of scope>
```

- `order_sha`: `sha256sum <order> | cut -c1-12`.
- `greenlit`: `yes` go given, `no` commander declined, `n/a` stopped before asking.
- After writing: one line to commander, report path and status.
- Stale `in-progress` (session died mid-run): commander deletes the attempt if nothing changed, else sets `status: partial` and adds BLOCKERS with the stop point. Then retry.

## Re-block

Officer blocks at assess ([officer.md](officer.md) step 6), latest attempt `blocked` or `partial`. Append to that attempt, no new one:
- Frontmatter: `order_rev`, `order_sha`, `date` set to this run. `rank`, `status`, `greenlit` kept.
- New REBLOCK section at the end. Newest holds current blockers. Earlier ones and BLOCKERS are history.

```markdown
# REBLOCK YYYY-MM-DD <your rank> rev <N>
- <what stopped>. Q: <exact question for commander>
```

## Dropped report

General writes it after commander greenlight, as next free attempt K. Never while latest attempt is `in-progress`: commander resolves the stale claim first.
- Frontmatter: `rank: general`, `greenlit: yes`, `order_rev` and `order_sha` at drop time.
- Body: only REASON and LEFT BEHIND.
- REASON: why commander dropped it. One fact per bullet.
- LEFT BEHIND: only when dropped from `partial`. One `file:line: change` per bullet, changes still in the repo.

```markdown
# REASON
- <why commander dropped the order>

# LEFT BEHIND
- file:line: change
```
