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
4. PR → CI five checks green → **merge commit** (not squash — the block's
   commits are the history) → dogfood rebuild.

## The gate checklist (quote output lines in reports)

```
make web-check        # svelte-check 0/0 + the full unit suite
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

- The semantic pins' faithfulness (spec → code, traced).
- The refusal matrices (role/lock/ownership gates — both the happy arm and
  every refusal).
- Removal thoroughness (grep to zero for anything the block retires).
- Honest-report semantics (set writes, idempotency, event discipline).
- House discipline: all-English, banned names, i18n parity, icon-first,
  the comment voice.
- Gate lines re-run by the reviewer, not trusted from the report.
