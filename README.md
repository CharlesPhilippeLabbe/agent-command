# ops

Claude Code plugin: a chain of command for agent work.

> [!WARNING]
> AI-generated. This is my first vibe-coded project: the skills, templates and the opsview viewer were written with Claude Code. Read it before you rely on it.

The **General** designs with you (the commander), has the **Scout** check the design against the code and the **Critic** review it, then plans and cuts orders. **Major**, **Captain** and **Soldier** execute orders, each asking you for one greenlight before executing, and file a report per attempt. The rank you give an order is how much control you keep over the AI's decisions.

Goals: predictable decisions under uncertainty, traceability across sessions, low token cost.

## Install

This repo is its own marketplace, `opossum` (see [.claude-plugin/marketplace.json](.claude-plugin/marketplace.json)). Add it once, then install the plugin:

```sh
claude plugin marketplace add CharlesPhilippeLabbe/agent-command
claude plugin install ops@opossum
```

Or inside a session: `/plugin marketplace add CharlesPhilippeLabbe/agent-command`, then `/plugin install ops@opossum`. Restart Claude Code; the skills show up as `/ops:general`, `/ops:scout`, `/ops:critic`, `/ops:major`, `/ops:captain`, `/ops:soldier`.

`--scope project` on `install` records the plugin in the repo's `.claude/settings.json` instead of your user settings.

## Update

```sh
claude plugin marketplace update opossum   # fetch the latest marketplace
claude plugin update ops@opossum           # install the new version
```

Restart Claude Code to load it. In a session, `/plugin` lists installed plugins and updates.

A new version is picked up only when `version` in [.claude-plugin/plugin.json](.claude-plugin/plugin.json) changes, so bump it with every release. `claude plugin tag` creates the matching `ops--v<version>` git tag.

## Try without installing

```sh
claude --plugin-dir /path/to/agent-command
```

## Ranks

| Skill | Implements | Gray zones | Greenlight shows | Takes orders of rank |
|---|---|---|---|---|
| `/ops:general` | never | questions everything, proposes alternatives, commander decides | one-line go: plan and orders are in opsview | n/a |
| `/ops:scout` | never | none: verdicts with file:line evidence, decides nothing | n/a | a plan in `design`, or an order draft |
| `/ops:critic` | never | none: one opsview note per finding | n/a | a plan in `design`, or an order draft |
| `/ops:major` | yes | resolves all, logs each | expected assumptions | major, captain, soldier |
| `/ops:captain` | yes | resolves minor, blocks on major or on more than 2 unlisted minor | gray zones and planned resolutions | captain, soldier |
| `/ops:soldier` | yes | none: any ambiguity blocks | objective, files, steps | soldier |

General treats your answers as input, not gospel: it questions them until you mark one definitive. The plan holds only what you decide: intent, interface, data, behavior. Implementation detail lives in the orders.

Rank is control: `soldier`, you take every decision. `captain`, the AI takes minor ones. `major`, the AI decides and you don't mind. Order drafts start at `major`; each refinement round pins one level more, until you stop it. Block risk is never a reason for a rank: the scout checks the code before anything is decided.

Officers never refuse on bookkeeping. Wrong rank, dependency not done, order unchanged since the last attempt, dropped without a new rev, a draft, a stale `in-progress` claim: each shows as a warning in the greenlight, and your go overrides it. Plans and orders are aerated, one subject per line, so a human can review them in opsview.

## Workflow

1. `/ops:general <feature>`: codename, interview, recon. The General writes the design into `plan.md` (`status: design`): goal, assumptions, uncertainties, alternatives.
2. Scout: `/ops:scout <op>`, or the General spawns it. It writes verdicts, friction points and new uncertainties into the plan.
3. Design rounds: you pick alternatives and friction, the General drafts decisions, lists its 3 weakest points, the critic (`/ops:critic <op>`) leaves notes, you leave notes in [opsview](viewer/). Repeat until you approve the plan.
4. Orders are drafted at `major` (`status: draft`). Refinement rounds pin one level more per order until you stop. Then you issue them.
5. Hand each order to its rank, in any session: `/ops:captain .saved/ops/vacuum-robot/orders/02-schedule.md`. Officer shows warnings, asks for go, executes, reports.
6. `/ops:general vacuum-robot`: debrief. Raises blockers and assumptions; ratified assumptions become plan decisions; answers go into the orders (rev bumped); follow-up orders go through drafting and refinement; dropped orders get a report from General; open notes get answered and resolved.
7. Retry a revised order with the same handoff command. A `blocked` or `partial` order also retries unchanged under `/ops:captain` or `/ops:major` when its blockers fit that rank.

## Files (in the target repo)

```
.saved/ops/<operation>/
  plan.md
  orders/NN-<slug>.md       # edited only by commander or General; status: draft while refined, rev bumped once issued, logged in plan
  reports/NN-<slug>.K.md    # attempt K, one per run; only an in-progress attempt is overwritten, by its final report; a re-block appends to the latest blocked or partial attempt
  notes/<target>.<hex>.md   # commander notes from opsview, critic notes (by: critic); open until General resolves
```

`<operation>` is a two-word kebab-case codename only remotely related to the feature, e.g. `vacuum-robot` for an auto-clean feature.

Report statuses: `in-progress` (written at go, replaced by the final report; claims the order), `done`, `partial`, `blocked`, `refused` (you declined the greenlight), `dropped` (written by General at debrief, commander's decision, carries the reason; officers warn until it gets a new rev).

Orders carry `risk: low|medium|high` and, from `medium` up, a RISKS section; officers show both in the greenlight.

Formats in [skills/shared/](skills/shared/): [plan](skills/shared/plan.md), [order](skills/shared/order.md), [report](skills/shared/report.md), [note](skills/shared/note.md), [style](skills/shared/style.md), shared [officer procedure](skills/shared/officer.md).

## Viewer

[viewer/](viewer/) holds `opsview`, a local dashboard of every operation that writes only notes: boards by dependency, order and report panels, docs browser. See [viewer/README.md](viewer/README.md).
