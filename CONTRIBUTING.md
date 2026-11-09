# Contributing to lumemor

lumemor is a self-hosted family photo & video library: a Go single binary
(embedding the web) + one PostgreSQL container. Every deployment serves one
family; the product serves families worldwide.

## Read first

- **`AGENTS.md`** — the entry point (written for agents and humans alike):
  the sources of truth, the iron rules, the workflow shape.
- **`docs/rules/`** — the per-language conventions (Go, web, commits &
  process). `CODEMAP.md` maps the repository.
- The design repository's specs are the semantic authority for every pinned
  surface — read the relevant spec before changing a pinned behavior.

## The iron rules (violations close the PR)

- **All-English repository** — code, comments, commits. Reference-project
  names never appear; write "the reference project" when comparing.
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

Commits: conventional, present tense, one decision per commit. PRs: one
focused block; the review checks the semantic pins, the refusal matrices,
the removal thoroughness, and re-runs the gates.

## Non-code contributions

i18n translations, documentation, and dogfooding reports are contributions
too — open an issue describing what you found. (The i18n pair `en` +
`zh-Hans` is the current surface; more locales follow the same parity
discipline.)
