#!/usr/bin/env python3
"""Consistency checks for the flat JSON translation files in i18n/.

The repository's translation conventions (see i18n/README.md):

  1. Every translation file is a flat JSON object: no nesting, every
     value is a string, and no key appears twice.
  2. Keys are lowercase snake_case identifiers matching ^[a-z0-9_]+$.
     Error-code keys are the error code lowercased
     (SETUP_ALREADY_COMPLETED -> setup_already_completed).
  3. en.json is the source of truth. Every other language file must
     carry exactly the same key set: missing and unexpected keys are
     both violations. (When more community-maintained languages land,
     this can soften to a coverage threshold; for now en and zh-Hans
     are both fully maintained, so exact equality is required.)
  4. ICU message placeholders ({name}) in a translation must match the
     placeholder set of the English string for the same key. Only
     simple {name} placeholders are compared.

Every violation is printed with the file it belongs to and makes the
script exit with status 1. With no violations a one-line summary is
printed and the script exits 0.

The script is pure standard-library Python 3 and expects to live in
scripts/ next to the i18n/ directory of the repository.
"""

import json
import re
import sys
from pathlib import Path

I18N_DIR = Path(__file__).resolve().parent.parent / "i18n"
SOURCE_FILE = "en.json"
KEY_PATTERN = re.compile(r"^[a-z0-9_]+$")
PLACEHOLDER_PATTERN = re.compile(r"\{([A-Za-z0-9_]+)\}")


def load_file(path: Path, violations: list) -> dict:
    """Parse one translation file, checking structure and key naming.

    Returns the key-to-string mapping. Files that cannot be parsed or
    that are not flat objects yield an empty mapping; every problem is
    appended to violations.
    """
    try:
        text = path.read_text(encoding="utf-8")
    except OSError as err:
        violations.append(f"{path.name}: cannot read file ({err})")
        return {}

    def reject_duplicates(pairs):
        seen = set()
        for key, _ in pairs:
            if key in seen:
                violations.append(f"{path.name}: duplicate key {key!r}")
            seen.add(key)
        return dict(pairs)

    try:
        data = json.loads(text, object_pairs_hook=reject_duplicates)
    except json.JSONDecodeError as err:
        violations.append(f"{path.name}: invalid JSON ({err})")
        return {}

    if not isinstance(data, dict):
        violations.append(f"{path.name}: top level must be a JSON object")
        return {}

    strings = {}
    for key, value in data.items():
        if not KEY_PATTERN.match(key):
            violations.append(
                f"{path.name}: key {key!r} must be lowercase snake_case "
                f"({KEY_PATTERN.pattern})"
            )
        if isinstance(value, str):
            strings[key] = value
        else:
            violations.append(
                f"{path.name}: value for {key!r} must be a string "
                "(translations must stay flat, no nesting)"
            )
    return strings


def check_against_source(name: str, table: dict, source: dict, violations: list) -> None:
    """Compare one language file with the source language file."""
    for key in sorted(set(table) - set(source)):
        violations.append(
            f"{name}: key {key!r} does not exist in {SOURCE_FILE}"
        )
    for key in sorted(set(source) - set(table)):
        violations.append(
            f"{name}: key {key!r} is missing ({SOURCE_FILE} is the source of truth)"
        )
    for key in sorted(set(table) & set(source)):
        expected = sorted(PLACEHOLDER_PATTERN.findall(source[key]))
        actual = sorted(PLACEHOLDER_PATTERN.findall(table[key]))
        if actual != expected:
            violations.append(
                f"{name}: placeholders for key {key!r} do not match "
                f"{SOURCE_FILE} (expected {expected}, got {actual})"
            )


def main() -> int:
    if not I18N_DIR.is_dir():
        print(f"error: i18n directory not found: {I18N_DIR}", file=sys.stderr)
        return 1

    files = sorted(I18N_DIR.glob("*.json"))
    if not files:
        print(f"error: no translation files found in {I18N_DIR}", file=sys.stderr)
        return 1

    violations: list = []
    tables = {path.name: load_file(path, violations) for path in files}

    source = tables.get(SOURCE_FILE)
    if source is None:
        violations.append(f"{SOURCE_FILE}: missing, but it is the source language file")
        source = {}

    for name, table in tables.items():
        if name != SOURCE_FILE:
            check_against_source(name, table, source, violations)

    if violations:
        print(f"i18n check FAILED: {len(violations)} violation(s):")
        for violation in violations:
            print(f"  - {violation}")
        return 1

    languages = ", ".join(name.removesuffix(".json") for name in tables)
    print(f"i18n check OK: {len(tables)} file(s) [{languages}], {len(source)} key(s)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
