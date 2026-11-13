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
- <what a human must not miss: irreversible step, ordering dependency, easy-to-misread constraint>

# SCOPE
IN: exact files/dirs that may be created or edited
OUT: files/dirs/behaviors forbidden

# STEPS
1. <one outcome: exact files, functions, values. Behavior, not code>
   Why: <only when not obvious: what breaks without it, or what a later step needs>

# DONE WHEN
- `<command>` gives `<expected>`

# RISKS
- <what could break>: <how to notice, how to avoid>

# DECIDED
- <decision>: <reason>

# OPEN
- <gray zone commander delegated, with bounds>
```

Omit `depends`, KEY POINTS, RISKS, OPEN when empty. Never write "maybe", "probably", "if needed", "etc.", "or similar".

## Altitude

Steps at the level a competent engineer needs: the outcome, not the motions.
- One step per outcome: "Add token expiry check to `Authenticator`", not "open the file, find the method, add an if".
- Skip what an engineer does untold: opening files, formatting, imports, saving.
- Spell out only what cannot be inferred: a decision, a constraint, a trap, an exact value.
- Fewer than 3 steps is fine. Do not split to look thorough. Step restating code or DONE WHEN: delete.

## Code in orders

Officer writes the code. Orders describe behavior, inputs, outputs, files and functions. No code blocks, pseudocode or full signatures. Exceptions:
- Exact literal the code must contain (command, config key, error string, name): inline, one line.
- Snippet the commander wrote or asked for: quoted unchanged.

Behavior needs more than a line: make it a DONE WHEN check.

## DONE WHEN

One check per bullet. Checks outcome, not code shape.
- Prefer: tests pass, build passes, command output, file exists.
- Absence check fine: `grep -rn old_name src/` gives nothing.
- No exact positive counts (`grep -c X` gives `1`): imports, call sites, re-exports make them unpredictable.
- Check needing code not yet written: replace with a behavior check.

## Editing

Only commander or General edits an order, `rev` +1 each.
- General: same write adds plan LOG line `YYYY-MM-DD: order NN rev N: <change>. <why>.` and updates ORDERS Rev.
- Commander: `rev` +1 only. General logs it at next debrief.

### Ripple check

A change in one part can make another wrong. Before showing any General edit for greenlight:
- Name what changed: decision, name, path, interface, behavior, or output another order relies on.
- Search plan and every order not `done` for it.
- Each hit: proposed edit, or no change with why. No hits: "ripple: none".
- Restate affected text in full in each order, never by ID reference, so each order reads alone.

## Gray zones

- **Minor**: naming, file-local structure, formatting, choice between equivalent idioms already in codebase, small gap fully inside SCOPE IN.
- **Major**: public interface, API contract, data model or schema, persisted data, dependencies, security, behavior visible outside SCOPE, SCOPE boundaries, conflict between order and `plan.md`.

## Risk

How bad a wrong execution is, independent of rank. `medium` and `high` need RISKS.
- `low`: local to SCOPE IN, easy to revert, DONE WHEN catches failure.
- `medium`: behavior change visible outside SCOPE IN, or failure DONE WHEN may miss.
- `high`: persisted data, security, public interface, deploys, or hard to revert.

## Rank by gray zones

Soldier default. Captain or major only with OPEN items the commander delegated ([general](../general/SKILL.md) Planning step 6).

| Rank | OPEN | Allowed gray zones |
|---|---|---|
| soldier | omitted | none |
| captain | 0-2 items | minor only |
| major | any | minor and major, except SCOPE boundaries |
