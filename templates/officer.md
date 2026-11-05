# Officer procedure

Shared by soldier, captain, major. Rank-specific rules are in each SKILL.md and override nothing here except where named.

Formats: [order.md](order.md), [report.md](report.md), [style.md](style.md).

Argument: path to order file.

## Procedure

1. **Read** order. Read `plan.md` for context. Order conflicting with `plan.md` is a major gray zone.
2. **Claim.** Latest attempt has status `in-progress`: another session holds the order. Tell commander its report path. Stop. Write no report.
3. **Rank.** Order rank not in your Takes list: `refused` ("order requires <rank>"). Stop.
4. **Dependencies.** For each `depends` NN: latest attempt report of NN must have `status: done`. Else `refused`, naming each missing dependency with its status (a `dropped` one reads as dropped). Stop.
5. **Retry.** If latest attempt has status `blocked`, `partial`, `done` or `dropped`, compare with it:
   - Same `rev`, same `order_sha`: `refused` ("no new rev"; latest `dropped`: "order dropped, no new rev"). Stop.
   - Same `rev`, different `order_sha`: `refused` ("order edited without rev bump"). Stop.
6. **Assess** before touching anything, per rank (SKILL.md: Block before go). Blocked: `blocked` report, exact question per item. Change nothing. Stop.
7. **Greenlight.** Show commander, terse: OBJECTIVE, `risk` and RISKS, files to touch, numbered STEPS, plus rank extras (SKILL.md: Greenlight adds). Ask "go?". Only a clear go starts work. Anything else: `refused`, quote reply. Stop. A reply that changes the order does not count; order changes go through commander or General editing the order. This is the only question you ask.
8. **Start.** On go, before any change: write attempt K report, frontmatter only, `status: in-progress`, `greenlit: yes`. Attempt K file already exists: another session claimed it first. Stop. Change nothing.
9. **Execute** STEPS in order, inside SCOPE IN. Gray zone mid-way: per rank (SKILL.md: Mid-way). On stop: leave code consistent. Something changed: `partial` with completed steps and exact stop point. Nothing changed: `blocked`.
10. **Verify.** Run every DONE WHEN check. Failure: per rank (SKILL.md: Verify fail). Any check still failing: `partial`. Record output verbatim.
11. **Report** per [report.md](report.md). Started: overwrite your `in-progress` attempt K with the final report. Stopped before step 8: new attempt K.

## Rules

- Never edit order or `plan.md`.
- Never reopen DECIDED.
- Never touch anything outside SCOPE IN.
- No destructive or irreversible action unless a STEP names it.
- Never commit or push unless a STEP says so.
- Never write a `dropped` report. Only General does.
- No scope expansion. Useful extras go to SUGGESTIONS.
- Chat terse, per [style.md](style.md).
