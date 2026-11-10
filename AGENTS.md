# AGENTS.md — lumemor

The entry point for agents AND humans working in this repository: what it
is, where its rules live, and the gates every change must pass.

## What this repository is

lumemor is a self-hosted family photo & video library: a Go single binary
(embedding the web app) plus one PostgreSQL container. Every deployment
serves one family; the product serves families worldwide. The behavior
contract lives in the code and its tests — `openapi/lumemor.yaml`, the
handlers, and the suites are the authority for what the product does.

## Where things live

- **`CODEMAP.md`** — the repository map: one page per area, the file that
  answers "where does this change go".
- **`docs/rules/`** — the repo's own coding conventions (Go, web, commits
  & process), stated in full.
- **`Makefile`** — the gate commands (`make web-check`, `make docker-lint`,
  `make docker-test`, `make api-check`, `python3 scripts/check_i18n.py`).
- **`.superpowers/sdd/<block>/`** — the per-block task briefs and reports
  (working artifacts, historical once the block merges).

## The iron rules (violations close the PR)

- **All-English repository**: code, comments, commits, and task artifacts.
  **Exemption classes**: the zh-Hans locale's values (`i18n/zh-Hans.json`
  is Chinese by definition); cited display names inside contract
  descriptions (e.g. the sync album's localized name). Reviews scan the
  added lines for CJK and judge the exemptions.
- **No other-project comparisons**: comparisons to other photo projects
  stay out of the code, comments, and commits entirely — this repo states
  its own decisions and their reasons, never borrowed authority.
- **Global product**: no default may assume a single market, language, or
  region. Chinese is a first-class citizen, not the constraint.
- **Contract-first**: edit `openapi/lumemor.yaml` by hand, regenerate, and
  end with a clean-tree `make api-check` (zero drift). sqlc queries change
  in the same loop. Migrations come in up+down pairs with honest comments.
- **The e2e hard rule**: any diff touching `web/e2e/**` or product code an
  e2e scene exercises must run the affected spec against a real stack
  before task close — ONE FRESH STACK LIFETIME PER SPEC (each spec's
  fixture bootstraps its own first-run admin; two specs on one lifetime
  answer 409).
- **Gates before close**: the five checks above, on the final tree, quoted
  in the task report.

## The workflow shape

Work lands serially on one branch per block: task briefs dispatch to
implementer agents, each writing a report; ONE independent review at the
end of the branch; then PR → CI (5 checks) → merge commit (not squash —
the block's commits are the history) → dogfood rebuild.

## Concurrent sessions

When another agent works this tree: never `git add -A`; explicit paths
only; a clean-tree `make api-check` before closing any task.
