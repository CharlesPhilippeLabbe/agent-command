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
- `active`: at least one report exists.
- `closed`: all orders `done`, goal met, commander greenlit closing.

Order status in ORDERS: `pending` (no report yet), else status of the latest report (`in-progress` while an officer executes).

DECISIONS Source: `plan` (decided while planning), `commander` (decided at debrief), or `NN.K` (assumption from report attempt K of order NN, ratified by commander).

Omit RISKS if none.
