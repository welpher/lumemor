# i18n

Flat JSON translation files. `en.json` is the source language; `zh-Hans.json` is officially maintained by the project.

- Keys are stable snake_case IDs; never rename keys, only change values
- Error-code keys mirror the `Error.code` enum in `openapi/lumemor.yaml`: the key is the code lowercased (`AUTH_RATE_LIMITED` -> `auth_rate_limited`); CI cross-checks both directions, and non-error keys such as `app_name` are allowlisted in `scripts/check_i18n.py`
- Placeholders use ICU style (`{count}`, plurals)
- Shared by web and mobile from this directory
- Missing translations fall back to en
- Contribution guide: `docs/CONTRIBUTING_i18n.md` (coming soon)
