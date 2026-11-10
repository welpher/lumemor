# Contributing to lumemor

lumemor is a self-hosted family photo & video library: a Go single binary
(embedding the web) + one PostgreSQL container. Every deployment serves one
family; the product serves families worldwide.

## Read first

- **`AGENTS.md`** — the entry point (written for agents and humans alike):
  the iron rules, the workflow shape, the gate commands.
- **`docs/rules/`** — the repo's own conventions (Go, web, commits &
  process). `CODEMAP.md` maps the repository.

## The iron rules (violations close the PR)

- **All-English repository** — code, comments, commits. The exemption
  classes: the zh-Hans locale's values and cited display names inside
  contract descriptions.
- **No other-project comparisons** — comparisons to other photo projects
  stay out of the code, comments, and commits entirely; state the decision
  and its reason in this repo's own words.
- **Global product** — no default assumes a single market, language, or
  region.
- **Contract-first** — API changes start in `openapi/lumemor.yaml`, then
  regenerate; a clean-tree `make api-check` ends with zero drift.

## Before you open the PR

Run the gates and make them green:

```
make web-check        # svelte-check 0/0 + the unit suite
make docker-lint      # the Go lint layer
make docker-test      # the server suite
make api-check        # regeneration determinism (zero drift)
python3 scripts/check_i18n.py
```

If your diff touches `web/e2e/**` or code an e2e scene exercises, run the
affected spec against a real stack (one fresh stack lifetime per spec —
see AGENTS.md for the procedure and the trap).

Match the surrounding code's conventions — `docs/rules/` states them in
full; the machines enforce format and lint, the review checks the rest.
Commits: conventional, present tense, one decision per commit. PRs: one
focused block; the review checks the semantic pins, the refusal matrices,
the removal thoroughness, and re-runs the gates.

## Non-code contributions

i18n translations, documentation, and dogfooding reports are contributions
too — open an issue describing what you found. (The i18n pair `en` +
`zh-Hans` is the current surface; more locales follow the same parity
discipline.)
