# AGENTS.md — lumemor

The entry point for agents AND humans. The design/decision record lives in a
separate private repository (the specs are the semantic authority — read the
relevant spec before touching a pinned surface); this repository is the
implementation.

## Sources of truth

- **The design repo's specs** — semantic pins (web IA §7 chain, data model,
  tech stack). Conflicts between code and spec resolve in the spec's favor.
- **`Makefile`** — the gate commands (`make web-check`, `make docker-lint`,
  `make docker-test`, `make api-check`, `make check-i18n`).
- **`CODEMAP.md`** — the repository map.
- **`docs/rules/`** — the per-language code conventions (Go, web, commits).
- **`.superpowers/sdd/<block>/`** — the per-block task briefs and reports
  (working artifacts, historical once the block merges).

## The iron rules

- **All-English repository**: code, comments, commits, and task artifacts.
  Reference-project names never appear here (reviews grep for them; write
  "the reference project" when comparing). **Exemption classes** (the
  established interpretation): the zh-Hans locale's values (`i18n/zh-Hans.json`
  is Chinese by definition); cited display names inside contract descriptions
  (e.g. the sync album's localized name); task briefs/reports quoting a pin's
  text. The check: `git grep -iE 'immich|photoprism'` over the surface plus a
  CJK scan of added lines, with the exemptions judged by a human.
- **Global product**: every deployment serves one family, the product serves
  families worldwide. No default may assume a single market, language, or
  region. Chinese is a first-class citizen, not the constraint.
- **Contract-first**: edit `openapi/lumemor.yaml` by hand, regenerate, and
  end with a clean-tree `make api-check` (zero drift). sqlc queries change in
  the same loop. Migrations come in up+down pairs with honest comments.
- **The e2e hard rule**: any diff touching `web/e2e/**` or product code an
  e2e scene exercises must run the affected spec against a real stack before
  task close — ONE FRESH STACK LIFETIME PER SPEC (each spec's fixture
  bootstraps its own first-run admin; two specs on one lifetime answer 409).
- **Gates before close**: the five checks above, on the final tree, quoted in
  the task report.

## The workflow shape

Blocks (milestone/extension) are implemented serially on one branch: task
briefs dispatch to implementer agents, each writing a report; ONE independent
review at the end of the branch; then PR → CI (5 checks) → merge commit.
Commit style: conventional, present tense, one decision per commit.

## Environment realities (restricted network)

- Proxy: `export https_proxy=http://127.0.0.1:7890 http_proxy=http://127.0.0.1:7890 all_proxy=socks5://127.0.0.1:7890`
- Build mirrors: `LUMEMOR_BUILD_GOPROXY='https://goproxy.cn,direct'
  LUMEMOR_BUILD_APT_MIRROR=mirrors.tuna.tsinghua.edu.cn
  LUMEMOR_BUILD_GEONAMES_MIRROR=http://host.docker.internal:8931`
- The dogfood stack: `~/lumemor-dogfood` (port 8244; STOP it before any
  throwaway stack uses 8244, restore it after).

## Concurrent sessions

When another agent works this tree: never `git add -A`; explicit paths only;
a clean-tree `make api-check` before closing any task.
