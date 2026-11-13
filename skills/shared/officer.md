# Officer procedure

Shared by soldier, captain, major. Each SKILL.md adds rank rules, overriding nothing here except where named. Formats: [order.md](order.md), [report.md](report.md), [style.md](style.md).

Argument: path to order file.

## Procedure

1. **Read** order and `plan.md`. Order conflicting with `plan.md`: major gray zone.
2. **Claim.** Latest attempt `in-progress`: another session holds the order. Tell commander its report path. Stop. Write no report.
3. **Rank** not in your Takes: `refused` ("order requires <rank>"). Stop.
4. **Dependencies.** Each `depends` NN needs latest attempt `done`. Else `refused`, naming each with its status. Stop.
5. **Retry.** Latest attempt `blocked`, `partial`, `done` or `dropped`: first matching rule applies.
   - `dropped`, same `rev`: `refused` ("order dropped, no new rev"). Stop.
   - `blocked` or `partial`, you are captain or major: proceed, order changed or not. Its BLOCKERS, newest REBLOCK and failed VERIFY are gray zones for step 6.
   - Same `order_sha`: `refused` ("order unchanged"). Stop.
   - Different `order_sha`: proceed, `rev` bumped or not.
6. **Assess** before touching anything (SKILL.md: Block before go). Blocked: exact question per item. Change nothing. Stop. Latest attempt `blocked` or `partial`: append a [re-block](report.md#re-block). Else: new `blocked` attempt.
7. **Greenlight.** Show, terse: OBJECTIVE, `risk` and RISKS, files to touch, numbered STEPS, rank extras (SKILL.md: Greenlight adds). Ask "go?", your only question. Only a clear go starts work. Anything else, a reply changing the order included: `refused`, quote reply. Stop. Order changes go through commander or General.
8. **Start.** On go, before any change: write attempt K, frontmatter only, `status: in-progress`, `greenlit: yes`. File already exists: another session claimed it. Stop. Change nothing.
9. **Execute** STEPS in order, inside SCOPE IN. Gray zone: SKILL.md: Mid-way. On stop, leave code consistent. Something changed: `partial`, with completed steps and exact stop point. Nothing changed: `blocked`.
   - Latest attempt `partial`: resume. Confirm code matches its DONE, continue from its stop point. Stop again: `partial`.
10. **Verify.** Run every DONE WHEN check, record output verbatim. Failure: SKILL.md: Verify fail. Still failing: `partial`.
    - Check wrong, code meets OBJECTIVE and STEPS (e.g. expects 1 match, code has import plus use): `check defect` in VERIFY, with why. Never edit the check. Soldier: still `partial`. Captain or major: verify its intent another way, log in ASSUMPTIONS. Intent met counts as passing. Captain: ambiguous intent is major, block.
11. **Report** per [report.md](report.md). Started: overwrite your `in-progress` attempt. Blocked at step 6: per step 6. Other stop before step 8: new attempt K.

## Rules

- Never edit order or `plan.md`. Never reopen DECIDED.
- Never touch anything outside SCOPE IN. No scope expansion: extras go to SUGGESTIONS.
- No destructive or irreversible action, commit or push unless a STEP names it.
- Never write a `dropped` report.
- Chat terse ([style.md](style.md)).
