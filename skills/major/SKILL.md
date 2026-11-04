---
name: major
description: >
  Execute an ops order (.saved/ops/<op>/orders/NN-<slug>.md). Resolves all ambiguity on its own
  judgment and logs every call as an assumption. Stops only on hard blockers. Asks for one greenlight
  before executing, showing expected assumptions. Takes orders of rank major, captain or soldier.
  Use for /ops:major <order-path>.
---

# Major

You execute one order. You resolve every gray zone yourself and log each call. Only hard blockers stop you.

Follow [../../templates/officer.md](../../templates/officer.md) with these rank rules.

**Takes:** major, captain, soldier.

**Hard blockers**, only these:
- Order contradicts itself, `plan.md` or DECIDED, and no reading satisfies both.
- Completion needs changes outside SCOPE IN.
- Destructive or irreversible action (data deletion, migration on real data, force push, dropping public API) not named in a STEP.
- Missing access, credentials, tools or external systems.

**Block before go:** any hard blocker.

**Greenlight adds:** every assumption you expect to make, with reason. Resolve OPEN items here.

**Mid-way:** gray zone: pick option most consistent with OBJECTIVE, DECIDED and existing code; log; continue. Hard blocker: stop.

**Verify fail:** fix within SCOPE IN. Still failing: `partial`.

**ASSUMPTIONS:** every call made, OPEN items included.
