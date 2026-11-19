# Plan format

Path: `.saved/ops/<op>/plan.md`. Style: [style.md](style.md), aerated layout.

Writers:
- General: everything. First `design` draft only once the commander said ready to draft, then drafts freely while `design`. Approval only after commander greenlight.
- Scout: only verdict lines under ASSUMPTIONS and UNCERTAINTIES, new FRICTION items, new UNCERTAINTIES, one LOG line per run. Never GOAL, DECISIONS, ORDERS.

What belongs here: what the commander decides. Intent, interface, data, behavior. Implementation detail (names, helpers, test cases, algorithms) belongs in orders.

```markdown
---
op: <op>
status: design|approved|issued|active|closed
created: YYYY-MM-DD
---
# GOAL

**Problem**
- ...

**End state**
- ...

**Success**
- ...

---

# CONSTRAINTS
- Hard limits from commander.

# OUT OF SCOPE
- ...

---

# ASSUMPTIONS

### A1 <title>
- <what we believe, one fact>
- Scout: holds|false|partial. <evidence, file:line>

# UNCERTAINTIES

### U1 <title>
- <open question>
- Scout: <answer>. <evidence, file:line>

# FRICTION

### F1 <title>
- Where: file:line
- Work around: <how, cost>
- Modify existing code: <how, cost>
- Pick: <commander's pick, or open>

# ALTERNATIVES

### <decision topic>
- <option>: <pros>. <cons>.
- <option>: <pros>. <cons>.
- Recommendation: <option>. <why>.

---

# DECISIONS

### D1 <title>
- Chosen: ...
- Rejected: <option>. <why>.
- Why: ...
- Source: plan

---

# ORDERS
| NN | Slug | Rank | Risk | Rev | Depends | Status |
|---|---|---|---|---|---|---|
| 01 | ... | major | low | 1 | - | draft |

# RISKS
- ...

# LOG
- YYYY-MM-DD: <change>. <why>.
```

## Status

- `design`: being designed, scouted, drafted, criticised. Before it exists, design is a discussion in chat. Written once the commander says ready to draft. General then writes drafts without greenlight.
- `approved`: commander greenlit the plan. ALTERNATIVES are gone, each turned into a DECISIONS block.
- `issued`: orders issued.
- `active`: a report exists.
- `closed`: all orders `done` or `dropped`, goal met or abandoned, commander greenlit. All `dropped` only when commander says so.

After a scout run the plan stays `design` for another round, or moves to `approved` on greenlight. No separate scouted status.

## Sections

- ASSUMPTIONS, UNCERTAINTIES: one block per item. The scout adds its verdict line under each. `false` or `partial` sends the plan to another design round.
- FRICTION: added by the scout where existing code resists the change. Commander picks. "Modify existing code" goes straight into the orders' STEPS.
- ALTERNATIVES: only while `design`.
- DECISIONS: one block per decision. Rejected and Why omitted when none.
  - Current truth only: what is decided now.
  - A changed decision is edited in place, never superseded by a new block. Its LOG line names the old and the new choice.
  - Source: `plan` (planning), `commander` (debrief), or `NN.K` (assumption from attempt K of order NN, ratified by commander).
- ORDERS Status: `draft` while the order is a draft, else `pending` without report, else latest report's status.
- LOG: one line per draft round, per scout run, and per write after approval.

Omit ASSUMPTIONS, UNCERTAINTIES, FRICTION, ALTERNATIVES, RISKS when empty.
