# ops

Claude Code plugin: a chain of command for agent work.

> [!WARNING]
> AI-generated. This is my first vibe-coded project: the skills, templates and the opsview viewer were written with Claude Code. Read it before you rely on it.

The **General** plans with you (the commander) and cuts the plan into orders. **Major**, **Captain** and **Soldier** execute orders with decreasing tolerance for ambiguity, each asking you for one greenlight before executing, and file a report per attempt.

Goals: predictable decisions under uncertainty, traceability across sessions, low token cost.

## Install

This repo is its own marketplace, `opossum` (see [.claude-plugin/marketplace.json](.claude-plugin/marketplace.json)). Add it once, then install the plugin:

```sh
claude plugin marketplace add CharlesPhilippeLabbe/agent-command
claude plugin install ops@opossum
```

Or inside a session: `/plugin marketplace add CharlesPhilippeLabbe/agent-command`, then `/plugin install ops@opossum`. Restart Claude Code; the skills show up as `/ops:general`, `/ops:major`, `/ops:captain`, `/ops:soldier`.

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
| `/ops:general` | never | questions everything, proposes alternatives, commander decides | full plan / orders | n/a |
| `/ops:major` | yes | resolves all, logs each | expected assumptions | major, captain, soldier |
| `/ops:captain` | yes | resolves minor, blocks on major or on more than 2 unlisted minor | gray zones and planned resolutions | captain, soldier |
| `/ops:soldier` | yes | none: any ambiguity blocks | objective, files, steps | soldier |

General resolves every gray zone with the commander while planning, so orders default to `soldier`. A higher rank is issued only when the commander delegates a named gray zone, listed in OPEN. Plans and orders keep one subject per line so a human can review them.

## Workflow

1. `/ops:general <feature>`: interview, alternatives, codename, plan, orders.
2. Hand each order to its rank, in any session: `/ops:captain .saved/ops/vacuum-robot/orders/02-schedule.md`. Officer checks, asks for go, executes, reports.
3. Review plan, orders and reports in [opsview](viewer/), and tap a block to leave a note. Notes wait in `notes/` for the next debrief.
4. `/ops:general vacuum-robot`: debrief. Raises blockers and assumptions; ratified assumptions become plan decisions; answers go into the orders (rev bumped); follow-up orders issued; dropped orders get a report from General; open notes get answered and resolved.
5. Retry a revised order with the same handoff command.

## Files (in the target repo)

```
.saved/ops/<operation>/
  plan.md
  orders/NN-<slug>.md       # edited only by commander or General; rev bumped, logged in plan
  reports/NN-<slug>.K.md    # attempt K, one per run; only an in-progress attempt is overwritten, by its final report
  notes/<target>.<hex>.md   # commander notes from opsview; open until General resolves
```

`<operation>` is a two-word kebab-case codename only remotely related to the feature, e.g. `vacuum-robot` for an auto-clean feature.

Report statuses: `in-progress` (written at go, replaced by the final report; claims the order), `done`, `partial`, `blocked`, `refused`, `dropped` (written by General at debrief, commander's decision, carries the reason; officers refuse the order until it gets a new rev).

Orders carry `risk: low|medium|high` and, from `medium` up, a RISKS section; officers show both in the greenlight.

Formats in [templates/](templates/): [plan](templates/plan.md), [order](templates/order.md), [report](templates/report.md), [note](templates/note.md), [style](templates/style.md), shared [officer procedure](templates/officer.md).

## Viewer

[viewer/](viewer/) holds `opsview`, a local dashboard of every operation that writes only notes: boards by dependency, order and report panels, docs browser. See [viewer/README.md](viewer/README.md).
