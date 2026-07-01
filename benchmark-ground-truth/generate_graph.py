#!/usr/bin/env python3
"""Generate the deterministic DOT graph used by Voyantclair benchmark CI."""

from __future__ import annotations

import argparse
import json
from pathlib import Path

DIRECTORY = Path(__file__).resolve().parent
OUTPUT = DIRECTORY / "rest-graph.dot"


def render() -> str:
    document = json.loads((DIRECTORY / "rest-edges.yaml").read_text(encoding="utf-8"))
    lines = ["digraph train_ticket {", "  rankdir=LR;"]
    lines.extend(f'  "{source}" -> "{target}";' for source, target in sorted(document["edges"]))
    lines.append("}")
    return "\n".join(lines) + "\n"


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--check", action="store_true")
    arguments = parser.parse_args()
    expected = render()
    if arguments.check:
        actual = OUTPUT.read_text(encoding="utf-8") if OUTPUT.exists() else ""
        if actual != expected:
            raise SystemExit("rest-graph.dot is stale; run benchmark-ground-truth/generate_graph.py")
    else:
        OUTPUT.write_text(expected, encoding="utf-8")


if __name__ == "__main__":
    main()
