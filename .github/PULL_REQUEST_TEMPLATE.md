## What

## Why

## Checks

- [ ] `make ci-local` is green (the full cold gate: web, server, contract
      lints, sqlc drift, locales)
- [ ] A web-only change ran the fast four instead: `make web-check`,
      `make docker-lint`, `python3 scripts/check_i18n.py`, and `make
      api-check` on a clean tree
- [ ] User-facing strings live in both `i18n/en.json` and
      `i18n/zh-Hans.json`
- [ ] Contract changes update `openapi/lumemor.yaml` and commit the
      generated clients (`make api-gen`)
- [ ] Tests cover the behavior (unit and/or an e2e scene)
- [ ] No secrets, credentials or private photos in the diff or the logs

## Notes