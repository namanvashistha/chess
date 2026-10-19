#!/usr/bin/env python3
"""Builds the Android app's translations from the iPhone app's string catalog.

The iPhone catalog (ios/Chess/Localizable.xcstrings) is the one source of
truth for every sentence in both apps. This writes it, plus the few lines only
Android says (tools/extra-strings.json), to app/src/main/assets/strings.json as
{english: {language: translation}}, which L() reads at launch.

Run it from the repository root with `make android-strings`.
"""
import json
import pathlib

root = pathlib.Path(__file__).resolve().parents[2]
catalog = json.loads((root / "ios/Chess/Localizable.xcstrings").read_text())["strings"]
extra = json.loads((root / "android/tools/extra-strings.json").read_text())

table = {}
for key, entry in catalog.items():
    if entry.get("shouldTranslate") is False:
        continue
    found = {
        lang: loc["stringUnit"]["value"]
        for lang, loc in entry.get("localizations", {}).items()
        if "stringUnit" in loc and lang != "en"
    }
    if found:
        table[key] = found
table.update(extra)

out = root / "android/app/src/main/assets/strings.json"
out.write_text(json.dumps(table, ensure_ascii=False, sort_keys=True, indent=1) + "\n")
print(f"{len(table)} strings → {out.relative_to(root)}")
