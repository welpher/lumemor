# i18n

Flat JSON translation files. `en.json` is the source language; `zh-Hans.json` is officially maintained by the project.

- Keys are stable snake_case IDs; never rename keys, only change values
- Placeholders use ICU style (`{count}`, plurals)
- Shared by web and mobile from this directory
- Missing translations fall back to en
- Contribution guide: `docs/CONTRIBUTING_i18n.md` (coming soon)
