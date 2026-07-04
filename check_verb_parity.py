#!/usr/bin/env python3
"""check_verb_parity — the dual-navigability gate for the DAG essentials.

One source (dag-verbs.json) feeds two faces; this gate proves they stay in sync:
  1. MACHINE  — every tool id in the contract is registered in main.go, and vice versa.
  2. MAPPING  — every verb.enacts is either "essay-only" or a real tool id.
  3. HUMAN    — every contract verb is rendered on the built madeofdoors site
                (../madeofdoors/dist/index.html), when that sibling build is present.

stdlib-only (json + re) — the reason the contract is JSON, not YAML. Prints
`CONTRACT PASS` / `CONTRACT FAIL` and exits 0 / 1, matching the house
validate_facet.py posture. Wire into pre-push / CI in both repos.
"""
import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent                    # dag-essentials-mcp/
CONTRACT = ROOT / "dag-verbs.json"
MAIN_GO = ROOT / "main.go"
SITE = ROOT.parent / "madeofdoors" / "dist" / "index.html"  # sibling, optional


def main() -> int:
    problems = []

    contract = json.loads(CONTRACT.read_text(encoding="utf-8"))
    tool_ids = [t["id"] for t in contract["tools"]]
    tool_id_set = set(tool_ids)
    verbs = contract["verbs"]

    # 1. MACHINE — contract tool ids vs main.go registered Names
    go_src = MAIN_GO.read_text(encoding="utf-8")
    go_names = set(re.findall(r'mcp\.AddTool\(s, &mcp\.Tool\{Name:\s*"([^"]+)"', go_src))
    if tool_id_set - go_names:
        problems.append(f"tools in contract but NOT registered in main.go: {sorted(tool_id_set - go_names)}")
    if go_names - tool_id_set:
        problems.append(f"tools registered in main.go but NOT in contract: {sorted(go_names - tool_id_set)}")

    # 2. MAPPING — every verb.enacts resolves
    for v in verbs:
        e = v.get("enacts")
        if e != "essay-only" and e not in tool_id_set:
            problems.append(f"verb {v.get('verb')!r} enacts unknown tool {e!r}")

    # 3. HUMAN — contract verbs vs rendered site (only if the sibling build exists)
    if SITE.exists():
        html = SITE.read_text(encoding="utf-8")
        rendered = set(re.findall(r'<article class="verb reveal"><h3><span>([^<]+)</span>', html))
        contract_verbs = {v["verb"] for v in verbs}
        if contract_verbs - rendered:
            problems.append(f"verbs in contract but NOT rendered on site: {sorted(contract_verbs - rendered)}")
        if rendered - contract_verbs:
            problems.append(f"verbs rendered on site but NOT in contract: {sorted(rendered - contract_verbs)}")
        site_note = f"site: {len(rendered)} verbs rendered"
    else:
        site_note = f"site: SKIPPED (no build at {SITE})"

    print(f"contract: {len(tool_ids)} tools, {len(verbs)} verbs | main.go: {len(go_names)} tools | {site_note}")
    if problems:
        for p in problems:
            print(f"  FAIL: {p}")
        print("CONTRACT FAIL")
        return 1
    print("CONTRACT PASS")
    return 0


if __name__ == "__main__":
    sys.exit(main())
