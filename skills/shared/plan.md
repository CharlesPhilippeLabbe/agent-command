# Plan format

Path: `.saved/ops/<op>/plan.md`. Writer: General only, after commander greenlight. Style: [style.md](style.md).

```markdown
---
op: <op>
status: approved|issued|active|closed
created: YYYY-MM-DD
---
# GOAL
Problem, trigger, end state.

# CONSTRAINTS
Hard limits from commander.

# DECISIONS
| D | Decision | Chosen | Rejected (why) | Source |
|---|---|---|---|---|
| D1 | ... | ... | ... | plan |

# ORDERS
| NN | Slug | Rank | Risk | Rev | Depends | Status |
|---|---|---|---|---|---|---|
| 01 | ... | soldier | low | 1 | - | pending |

# RISKS
- ...

# LOG
- YYYY-MM-DD: <change>. <why>.
```

Plan status:
- `approved`: plan written, no orders yet.
- `issued`: orders written.
- `active`: a report exists.
- `closed`: all orders `done` or `dropped`, goal met or abandoned, commander greenlit. All `dropped` only when commander says so.

ORDERS Status: `pending` without report, else latest report's status.

DECISIONS Source: `plan` (planning), `commander` (debrief), or `NN.K` (assumption from attempt K of order NN, ratified by commander).

Omit RISKS if none.
