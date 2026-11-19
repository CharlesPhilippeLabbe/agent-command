# Style for ops files

Applies to plan, orders, reports, notes and chat. Token cost matters; ambiguity matters more.

- Terse. Fragments. No articles, filler, restating, summaries.
- Never cut: paths, names, values, commands, numbers, negations (not, never, only, except).
- A fragment with two readings becomes a full sentence.
- One term per thing. No invented abbreviations. No arrows.
- Omit empty sections. Missing section means none.

## No bare IDs

Never refer to a decision, assumption, uncertainty, friction item or order by its ID alone, in chat or in files. Content first, ID in parentheses at most.

Bad: "D23 still applies to 04."

Good: "Plan holds only intent, interface, data and behavior (D23). Still applies to the general-design order (04)."

## Layout

Plan and orders are read by a human in opsview: aerated. Reports, notes and chat stay terse and compact.

Plan and orders:
- `---` between major sections.
- Headings for each block a human reviews on its own: a decision, an assumption, a step.
- Short labeled lines (`**Where**: ...`, `**Title**: ...`) over walls of text.
- Blank line between blocks.

Formats: [plan.md](plan.md), [order.md](order.md#layout).

## Readability

Plan and orders are read by a human. Terse is fine; dense is not.

- One subject per line. One check, one action or one fact per bullet.
- No `;` joining different subjects. Split into bullets.
- No sentence joining unrelated things with "and". Split.
- Enumerations (tests, files, cases) are a bullet list, not a comma or semicolon run.
- State cause and effect: "X, so Y" or "Without X, Y breaks". Never leave it implied.
- Put a short label line over a group of related bullets.
- A fragment carries one subject.

Bad:

```
Tests in src/test/java/utils/authentication/: reset-purpose token rejected by Authenticator (AuthenticatorTest style) and by getRefreshToken; normal token still accepted; passwordFingerprint deterministic and differs from input.
```

Good:

```
**Where**: src/test/java/utils/authentication/, AuthenticatorTest style

**Cases**
- Authenticator rejects reset-purpose token.
- getRefreshToken rejects reset-purpose token.
- Authenticator and getRefreshToken still accept normal token.
- passwordFingerprint gives same output for same input.
- passwordFingerprint output differs from input.
```
