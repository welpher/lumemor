#!/usr/bin/env bash
# Negative-fixture tests for the error-code cross-check of
# scripts/check_i18n.py (check 5: the Error.code enum of
# openapi/lumemor.yaml and the i18n keys must agree in both
# directions).
#
# Every test copies the pristine files into a temporary directory,
# breaks exactly one of them, and expects the checker, pointed at the
# copy through its repo-root argument, to exit 1 with a message naming
# the broken bit. The real tree is never mutated. A pristine copy must
# pass first, so a fixture failure is always a failure of the check,
# not of the setup.
#
# The first failed assertion aborts with a non-zero exit status and a
# diagnostic message; the temporary directory is always removed. A
# passing run prints "I18N FIXTURES PASS" as its final line.
#
# Requirements: python3 (standard library only).

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
checker="$repo_root/scripts/check_i18n.py"

fail() {
    echo "I18N FIXTURES FAIL: $*" >&2
    exit 1
}

work_dir="$(mktemp -d)"
cleanup() {
    rm -rf "$work_dir"
}
trap cleanup EXIT

# Copy the pristine inputs into a scratch root the checker can be
# pointed at with its repo-root argument.
copy_tree() {
    local root="$1"
    mkdir -p "$root/i18n" "$root/openapi"
    cp "$repo_root/i18n/en.json" "$repo_root/i18n/zh-Hans.json" "$root/i18n/"
    cp "$repo_root/openapi/lumemor.yaml" "$root/openapi/"
}

# Run the checker against a fixture root; fail when it exits 0. The
# checker's output is echoed either way and must contain $2.
expect_failure() {
    local name="$1" root="$2" pattern="$3"
    echo "==> fixture: $name"
    if output="$(python3 "$checker" "$root" 2>&1)"; then
        fail "$name: checker exited 0, expected 1"
    fi
    printf '    %s\n' "$output"
    if ! printf '%s\n' "$output" | grep -Fq "$pattern"; then
        fail "$name: output does not mention '$pattern'"
    fi
}

echo "==> fixture: pristine copy passes"
copy_tree "$work_dir/pristine"
if ! output="$(python3 "$checker" "$work_dir/pristine" 2>&1)"; then
    printf '%s\n' "$output" >&2
    fail "pristine copy: checker must exit 0 on an unmodified tree"
fi
printf '    %s\n' "$output"

# Deleting the translation key from every language file isolates the
# forward direction of the cross-check: no other check may fire, so
# only the missing error code can be reported.
copy_tree "$work_dir/missing_key"
python3 - "$work_dir/missing_key" <<'PY'
import json
import sys
from pathlib import Path

root = Path(sys.argv[1])
for name in ("en.json", "zh-Hans.json"):
    path = root / "i18n" / name
    data = json.loads(path.read_text(encoding="utf-8"))
    del data["auth_rate_limited"]
    path.write_text(
        json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8"
    )
PY
expect_failure "deleted i18n key" "$work_dir/missing_key" \
    "has no translation key 'auth_rate_limited'"

# Renaming an enum value trips both directions: the renamed code has no
# translation key and the old key is orphaned.
copy_tree "$work_dir/renamed_code"
python3 - "$work_dir/renamed_code" <<'PY'
import sys
from pathlib import Path

path = Path(sys.argv[1]) / "openapi" / "lumemor.yaml"
text = path.read_text(encoding="utf-8")
assert "AUTH_RATE_LIMITED" in text
path.write_text(
    text.replace("AUTH_RATE_LIMITED", "AUTH_THROTTLED"), encoding="utf-8"
)
PY
expect_failure "renamed error code (missing key direction)" \
    "$work_dir/renamed_code" \
    "has no translation key 'auth_throttled'"
expect_failure "renamed error code (orphan key direction)" \
    "$work_dir/renamed_code" \
    "key 'auth_rate_limited' is not the lowercase form of any error code"

# Removing the enum block entirely must fail loudly instead of
# silently cross-checking zero codes.
copy_tree "$work_dir/no_enum"
python3 - "$work_dir/no_enum" <<'PY'
import sys
from pathlib import Path

path = Path(sys.argv[1]) / "openapi" / "lumemor.yaml"
text = path.read_text(encoding="utf-8")
block = """          enum:
            - SETUP_ALREADY_COMPLETED
            - VALIDATION_FAILED
            - INTERNAL
            - AUTH_FORBIDDEN
            - AUTH_INVALID_CREDENTIALS
            - AUTH_RATE_LIMITED
            - PAIRING_TOKEN_INVALID
            - ASSET_TOO_LARGE
            - ASSET_NOT_FOUND
            - ASSET_NOT_READY
            - USER_EMAIL_TAKEN
            - USER_NOT_FOUND
"""
assert block in text
path.write_text(text.replace(block, ""), encoding="utf-8")
PY
expect_failure "enum block removed" "$work_dir/no_enum" \
    "components.schemas.Error.properties.code.enum not found"

echo "I18N FIXTURES PASS"
