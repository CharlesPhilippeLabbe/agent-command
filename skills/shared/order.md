# Order format

Path: `.saved/ops/<op>/orders/NN-<slug>.md`. Style: [style.md](style.md), aerated layout.

```markdown
---
op: <op>
order: NN
slug: <slug>
rank: soldier|captain|major
risk: low|medium|high
rev: 1
depends: [NN, NN]
status: draft|issued
---
# OBJECTIVE

One sentence. End state, not activity.

# KEY POINTS

- <what a human must not miss: irreversible step, ordering dependency, easy-to-misread constraint>

---

# SCOPE

**IN**
- exact files/dirs that may be created or edited

**OUT**
- files/dirs/behaviors forbidden

---

# STEPS

## 1. <outcome>

**Where**: <file, function, section>

**Do**
- <one fact per bullet: behavior, values. Not code>

**Why**
- <only when not obvious: what breaks without it, or what a later step needs>

**Cases**
- <test case or edge case, one per bullet>

## 2. <outcome>

...

---

# DONE WHEN

- `<command>` gives `<expected>`

---

# RISKS

- <what could break>: <how to notice, how to avoid>

# DECIDED

- **<title>**: <decision>. <reason>.

# OPEN

- <gray zone left to the officer, with bounds>

# SCOUT

- <written by the order scout only>
```

- `status`: `draft` while the General refines the order, `issued` once handed out. Missing means `issued`. Officers warn on `draft`, never refuse ([officer.md](officer.md)).
- Omit `depends`, KEY POINTS, RISKS, OPEN, SCOUT when empty. Omit Why and Cases blocks when empty.
- Never write "maybe", "probably", "if needed", "etc.", "or similar".

## Layout

- `---` between major sections and between steps.
- One `##` heading per step, named by its outcome.
- Location on one line: `**Where**: location`.
- Labeled blocks under each step: Do, Why, Cases. One fact per bullet.
- DECIDED: one line per decision, `**Title**: description`. Restate the decision, never only its plan ID.

## Altitude

Steps at the level a competent engineer needs: the outcome, not the motions.
- One step per outcome: "Token expiry check in `Authenticator`", not "open the file, find the method, add an if".
- Skip what an engineer does untold: opening files, formatting, imports, saving.
- Spell out only what cannot be inferred: a decision, a constraint, a trap, an exact value.
- Fewer than 3 steps is fine. Do not split to look thorough. Step restating code or DONE WHEN: delete.
- How much is pinned depends on the refinement level ([rank](#rank-is-control)).

## Code in orders

Officer writes the code. Orders describe behavior, inputs, outputs, files and functions. No code blocks or pseudocode. Exceptions:
- Exact literal the code must contain (command, config key, error string, name): inline, one line.
- Signature, from captain level on, taken from code the scout or the General read: inline, one line.
- Snippet the commander wrote or asked for: quoted unchanged.

Behavior needs more than a line: make it a DONE WHEN check.

## Friction picks

A FRICTION item in `plan.md` where the commander picked "modify existing code": the change goes straight into STEPS, in the order that needs it. No separate refactor order.

## Order scout

Runs on a draft, only on commander's ok. Writes only the SCOUT section:
- Signatures of the functions the order touches or adds, from real code.
- Call flow: who calls what, file:line.
- DONE WHEN dry-check: per check, runnable now or not, expected output predictable or not, why.

The General folds SCOUT findings into STEPS and DONE WHEN at the next refinement round, then deletes the section.

## DONE WHEN

One check per bullet. Checks outcome, not code shape.
- Prefer: tests pass, build passes, command output, file exists.
- Absence check fine: `grep -rn old_name src/` gives nothing.
- No exact positive counts (`grep -c X` gives `1`): imports, call sites, re-exports make them unpredictable.
- Check needing code not yet written: replace with a behavior check.

## Editing

Only commander or General edits an order.
- Draft (`status: draft`): edited freely, `rev` stays 1. General adds one plan LOG line per refinement round.
- Issued: `rev` +1 per edit.
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

## Rank is control

Rank is how much control the commander keeps over the AI's decisions. Commander picks it. Never picked from block risk or pinning cost: block risk is the scout's job.
- `soldier`: commander takes every decision.
- `captain`: AI takes minor ones.
- `major`: AI decides, commander does not mind.

A draft starts at `major` and goes one level down per refinement round, until the commander stops it.

| Level | Order pins | Officer decides | OPEN |
|---|---|---|---|
| major | OBJECTIVE, SCOPE, DONE WHEN, KEY POINTS, plan decisions in DECIDED, outcome-level STEPS | minor and major gray zones, except SCOPE boundaries | any |
| captain | adds: STEPS per outcome, interfaces, signatures, data, friction picks | minor gray zones only | 0-2 items |
| soldier | adds: names, placement, test cases, exact values | nothing | omitted |
