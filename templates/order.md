# Order format

Path: `.saved/ops/<op>/orders/NN-<slug>.md`. Style: [style.md](style.md).

```markdown
---
op: <op>
order: NN
slug: <slug>
rank: soldier|captain|major
risk: low|medium|high
rev: 1
depends: [NN, NN]
---
# OBJECTIVE
One sentence. End state, not activity.

# KEY POINTS
- <thing a human must not miss: irreversible step, ordering dependency, easy-to-misread constraint>

# SCOPE
IN: exact files/dirs that may be created or edited
OUT: files/dirs/behaviors forbidden

# STEPS
1. One action per step. Exact files, functions, values.
   Why: <cause and effect: what breaks without it, or what a later step needs from it>

# DONE WHEN
- `<command>` gives `<expected>`
- <checkable condition, one subject>


# RISKS
- <what could break>: <how to notice, how to avoid>

# DECIDED
- <decision>: <reason>

# OPEN
- <gray zone commander explicitly delegated, with bounds>
```

Omit `depends` if none. Omit KEY POINTS if none. Omit RISKS if none. Omit OPEN if none. No "maybe", "probably", "if needed", "etc.", "or similar".

## Editing

Only commander or General edits an order. Officers never do. Each edit: `rev` +1.

General edit, same write: plan LOG line `YYYY-MM-DD: order NN rev N: <change>. <why>.` and plan ORDERS row Rev updated.

Commander edit: `rev` +1 only. Plan stays General-only; General logs it at next debrief.

## Gray zones

**Minor**: naming, file-local structure, formatting, choice between equivalent idioms already in codebase, small gap fully inside SCOPE IN.

**Major**: anything touching public interface, API contract, data model or schema, persisted data, dependencies, security, behavior visible outside SCOPE, SCOPE boundaries, or conflict between order and `plan.md`.

## Risk

How bad a wrong execution is, independent of rank:

- `low`: local to SCOPE IN, easy to revert, DONE WHEN catches failure.
- `medium`: behavior change visible outside SCOPE IN, or failure DONE WHEN may miss.
- `high`: persisted data, security, public interface, deploys, or hard to revert.

`medium` and `high` need RISKS.

## Rank by gray zones

Soldier is the default. Captain or major only when OPEN holds items the commander delegated.

| Rank | OPEN | Allowed gray zones |
|---|---|---|
| soldier | omitted | none |
| captain | 0-2 items | minor only |
| major | any | minor and major, except SCOPE boundaries |
