---
name: shared
description: >
  Shared formats for the ops skills: plan, order, report, note, officer, style. The general,
  major, captain and soldier skills link to these files. Not a task skill: never invoke directly.
user-invocable: false
disable-model-invocation: true
---

# Shared

Formats the ops skills link to. Each file is the only source for its format.

| File | Defines |
|---|---|
| [plan.md](plan.md) | `.saved/ops/<op>/plan.md`, written by the General |
| [order.md](order.md) | `orders/NN-<slug>.md`: rank, risk, steps, done-when |
| [report.md](report.md) | `reports/NN-<slug>.K.md`: officer attempt reports |
| [note.md](note.md) | `notes/*.md`: commander notes from opsview |
| [officer.md](officer.md) | Rules common to major, captain, soldier |
| [style.md](style.md) | Writing style for every ops file |
