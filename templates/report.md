# Report format

Path: `.saved/ops/<op>/reports/NN-<slug>.K.md`. K = attempt, starting at 1. Create `reports/` if missing. Never overwrite an earlier attempt, except your own `in-progress` attempt, which the final report replaces. Style: [style.md](style.md).

Always write a report, whatever the outcome. Only exception: order already `in-progress` in another session ([officer.md](officer.md) step 2).

Status:
- `in-progress`: greenlit, executing. Written at go, frontmatter only; claims the order. Never a final status.
- `done`: every STEP executed, every DONE WHEN check passes.
- `partial`: changes made, then stopped, or a DONE WHEN check fails.
- `blocked`: stopped before changing anything: ambiguity or hard blocker.
- `refused`: not taken: wrong rank, dependency not `done`, no new rev, order edited without rev bump, no greenlight.

```markdown
---
op: <op>
order: NN
slug: <slug>
attempt: K
order_rev: N
order_sha: <first 12 hex of sha256 of order file>
rank: <executing officer: soldier|captain|major>
status: in-progress|done|partial|blocked|refused
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
