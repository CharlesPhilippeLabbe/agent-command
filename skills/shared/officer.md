# Officer procedure

Shared by soldier, captain, major. Each SKILL.md adds rank rules, overriding nothing here except where named. Formats: [order.md](order.md), [report.md](report.md), [style.md](style.md).

Argument: path to order file.

## Procedure

1. **Read** order and `plan.md`. Order conflicting with `plan.md`: major gray zone.
2. **Warnings.** Never refuse on bookkeeping. Each check that hits becomes one warning line for the greenlight (step 7):
   - Latest attempt `in-progress`: "attempt K in-progress, stale?". Another session may hold the order.
   - Rank not in your Takes: "order is <rank>, I take <ranks>".
   - Each `depends` NN whose latest attempt is not `done`: "depends on NN, <status>".
   - Same `order_sha` as the latest attempt: "unchanged since attempt K, <status>".
   - Latest attempt `dropped`, same `rev`: "dropped, no new rev".
   - Order `status: draft`: "order is a draft".
3. **Takes.** Rank above your Takes: execute it with your own rank rules. Its OPEN items are gray zones for step 6.
4. **Dependencies.** Not `done`: execute on go anyway. Code they were to deliver missing: gray zone for step 6.
5. **Retry.** Latest attempt `blocked` or `partial`, you are captain or major: its BLOCKERS, newest REBLOCK and failed VERIFY are gray zones for step 6. Same `order_sha` and status `done`: nothing to change; say so in the greenlight.
6. **Assess** before touching anything (SKILL.md: Block before go). Blocked: exact question per item. Change nothing. Stop. Latest attempt `blocked` or `partial`: append a [re-block](report.md#re-block). Else: new `blocked` attempt.
7. **Greenlight.** Show, terse: warnings from step 2 first, then OBJECTIVE, `risk` and RISKS, files to touch, numbered STEPS, rank extras (SKILL.md: Greenlight adds). Ask "go?", your only question. A clear go starts work and overrides every warning. Anything else, a reply changing the order included: `refused`, quote reply. Stop. Order changes go through commander or General.
8. **Start.** On go, before any change: write attempt K (next free), frontmatter only, `status: in-progress`, `greenlit: yes`. A stale `in-progress` attempt the commander overrode stays as it is. File already exists at write time: another session claimed it this moment. Stop. Change nothing.
9. **Execute** STEPS in order, inside SCOPE IN. Gray zone: SKILL.md: Mid-way. On stop, leave code consistent. Something changed: `partial`, with completed steps and exact stop point. Nothing changed: `blocked`.
   - Latest attempt `partial`: resume. Confirm code matches its DONE, continue from its stop point. Stop again: `partial`.
10. **Verify.** Run every DONE WHEN check, record output verbatim. Failure: SKILL.md: Verify fail. Still failing: `partial`.
    - Check wrong, code meets OBJECTIVE and STEPS (e.g. expects 1 match, code has import plus use): `check defect` in VERIFY, with why. Never edit the check. Soldier: still `partial`. Captain or major: verify its intent another way, log in ASSUMPTIONS. Intent met counts as passing. Captain: ambiguous intent is major, block.
11. **Report** per [report.md](report.md). Started: overwrite your `in-progress` attempt. Overridden warnings in OVERRIDES. Blocked at step 6: per step 6. Declined at step 7: new `refused` attempt K.

## Rules

- Never edit order or `plan.md`. Never reopen DECIDED.
- Never touch anything outside SCOPE IN. No scope expansion: extras go to SUGGESTIONS.
- No destructive or irreversible action, commit or push unless a STEP names it.
- Never write a `dropped` report.
- Chat terse ([style.md](style.md)).
