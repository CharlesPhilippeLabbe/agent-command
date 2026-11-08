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
1. One outcome per step, not per keystroke. Name exact files, functions, values. Describe behavior; do not write the code.
   Why: <only when not obvious: what breaks without it, or what a later step needs from it>

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

## Altitude

Write each step at the level a competent engineer needs. Name the outcome, not the motions.

- One step per outcome, not per keystroke. "Add token expiry check to `Authenticator`", not "open the file, find the method, add an if, compare the timestamp".
- Skip anything a competent engineer does without being told: opening files, running the formatter, importing what the code uses, saving.
- Spell out only what the engineer cannot infer: a decision, a constraint, a trap, an exact value.
- Order with fewer than 3 steps is fine. Do not split a step to look thorough.
- A step that restates what the code or DONE WHEN already makes obvious: delete it.

## Code in orders

Orders say what to do, not how to code it. The officer writes the code.

- Describe behavior, inputs, outputs and the files and functions involved.
- No code blocks, pseudocode or full signatures.
- Exception: an exact literal the code must contain (command, config key, error string, name). Inline, one line.
- Exception: a snippet the commander wrote or asked for. Quote it unchanged.
- Need more than a line to pin down a behavior: state it as a DONE WHEN check instead.

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
