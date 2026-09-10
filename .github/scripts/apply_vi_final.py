#!/usr/bin/env python3
import base64
import json
import zlib
from pathlib import Path

ROOT = Path.cwd()
ARB = ROOT / "lib" / "l10n" / "arb"
EN_PATH = ARB / "app_en.arb"
VI_PATH = ARB / "app_vi.arb"

PAYLOAD = "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"

def main() -> None:
    data = json.loads(zlib.decompress(base64.b64decode(PAYLOAD)).decode("utf-8"))
    translations = data["translations"]
    overrides = data["overrides"]
    en = json.loads(EN_PATH.read_text(encoding="utf-8"))
    vi = json.loads(VI_PATH.read_text(encoding="utf-8"))
    en_keys = {k for k in en if not k.startswith("@")}
    vi_keys = {k for k in vi if not k.startswith("@")}
    missing = en_keys - vi_keys
    expected = set(translations)
    if missing != expected:
        raise SystemExit(f"Translation patch out of sync: missing_only={sorted(missing-expected)} extra_patch={sorted(expected-missing)}")

    for key in sorted(missing):
        vi[key] = translations[key]
        meta_key = "@" + key
        if meta_key in en:
            vi[meta_key] = en[meta_key]

    for key, value in overrides.items():
        if key not in en_keys:
            raise SystemExit(f"Override key no longer exists: {key}")
        vi[key] = value
        meta_key = "@" + key
        if meta_key in en and meta_key not in vi:
            vi[meta_key] = en[meta_key]

    problems = []
    for key in sorted(en_keys):
        if key not in vi:
            problems.append(f"missing key: {key}")
            continue
        meta = en.get("@" + key)
        placeholders = set(meta.get("placeholders", {})) if isinstance(meta, dict) and isinstance(meta.get("placeholders", {}), dict) else set()
        value = str(vi[key])
        for placeholder in placeholders:
            if "{" + placeholder not in value:
                problems.append(f"{key}: missing {{{placeholder}}}")
    if problems:
        raise SystemExit("\n".join(problems))

    VI_PATH.write_text(json.dumps(vi, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"Added {len(missing)} Vietnamese messages and refreshed {len(overrides)} existing labels.")

if __name__ == "__main__":
    main()
