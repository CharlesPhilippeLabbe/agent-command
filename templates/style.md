# Style for ops files

Applies to plan, orders, reports and officer chat. Token cost matters; ambiguity matters more.

- Ultra-terse. Fragments. No articles, filler, restating, summaries.
- Never cut: paths, names, values, commands, numbers, negations (not, never, only, except).
- A fragment with two readings becomes a full sentence.
- One term per thing. No invented abbreviations. No arrows.
- Omit empty sections. Missing section means none.

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
Tests in src/test/java/utils/authentication/, in AuthenticatorTest style:
- Authenticator rejects reset-purpose token.
- getRefreshToken rejects reset-purpose token.
- Authenticator and getRefreshToken still accept normal token.
- passwordFingerprint gives same output for same input.
- passwordFingerprint output differs from input.
```
