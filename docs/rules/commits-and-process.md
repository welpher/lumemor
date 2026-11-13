# Commits, PRs, and process

## Commits

Conventional style, present tense, one decision per commit:
`feat:`, `fix:`, `docs:`, `ci:`, `test:`. The subject states the decision,
the body (when needed) carries the reasoning. All-English (the iron rule).

## The PR flow

1. Branch per block (`e6-album-autonomy`, `web-interface-debt`); tasks land
   serially on it.
2. Each task: implement → gates on the final tree → report at
   `.superpowers/sdd/<block>/task-N-report.md` (the commit hash, the
   contract surface, the test pins with evidence, the gates' quoted output
   lines, every deviation with its reasoning).
3. ONE independent review per branch (findings with file:line, verdict,
   merge recommendation); fix loop if required.
4. PR → the machine gate **by tier** + the per-PR CI thread (openapi,
   i18n, web — the outside-in check) → **merge commit** (not squash —
   the block's commits are the history) → dogfood rebuild. **Feature
   tiers** (contract changes, server semantics, cross-surface work,
   new surfaces) run `make ci-local` — the full cold reproduction;
   **mechanical tiers** (visual sweeps, format/comment/docs/config,
   i18n-only) run the fast four (`web-check` + `docker-lint` +
   `api-check` + `check_i18n`) — the cold containers guard inputs
   (the lockfile, the server code, the contract) those diffs never
   touch. The tier call rides the plan's own review-tier pin; a
   mechanical block whose diff surprises into those inputs is
   reclassified and re-gated. The two heavy workflows (server, e2e)
   run on demand and on a monthly schedule, not per-PR; the
   acceptance tier keeps its own fresh-lifetime discipline.

## The gate checklist (quote output lines in reports)

```
make ci-local        # the feature-tier merge gate: the full set, cold
                     # (ephemeral web container, cache-less server
                     # container, codegen drift twice, locales)
                     # mechanical tiers run the fast four below
make web-check        # format:check + eslint + svelte-check 0/0 + the unit suite
make docker-lint      # gofmt/vet/golangci in the container
make docker-test      # the full server suite
make api-check        # clean-tree regeneration: zero drift
python3 scripts/check_i18n.py
```

Plus the **e2e hard rule** when the diff touches `web/e2e/**` or exercised
product code: the affected specs run against a real stack, one fresh
lifetime per spec (the 409 trap), the dogfood stopped first and restored
last.

## What reviews check (the recurring focus)

- The semantic pins' faithfulness (contract → code, traced).
- The refusal matrices (role/lock/ownership gates — both the happy arm and
  every refusal).
- Removal thoroughness (grep to zero for anything the block retires).
- Honest-report semantics (set writes, idempotency, event discipline).
- House discipline: all-English (with the AGENTS.md exemption classes),
  banned names, i18n parity, icon-first, the comment voice.
- Gate lines re-run by the reviewer, not trusted from the report.

The house-discipline greps a review runs on the diff's added lines:

```
# the no-other-project-comparison discipline: comparisons to other photo
# projects stay out of the added lines entirely (no names, no euphemisms,
# no borrowed authority); the review greps for them and expects zero
# added-line hits
# CJK scan of added lines -- only the exemption classes may carry it
```
