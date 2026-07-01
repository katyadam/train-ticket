#!/usr/bin/env python3
"""Validate the machine-readable Voyantclair oracle without third-party packages."""

from __future__ import annotations

import json
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parents[1]
GROUND_TRUTH = ROOT / "benchmark-ground-truth"
SERVICE_PATTERN = re.compile(r"ts-[a-z0-9-]+-service")
REWRITES = {
    "ts-route-plan-service": ("go", {"ts-route-service", "ts-travel-service", "ts-travel2-service"}),
    "ts-station-service": ("go", set()),
    "ts-travel-plan-service": ("python", {"ts-route-plan-service", "ts-seat-service", "ts-train-service", "ts-travel-service", "ts-travel2-service"}),
    "ts-consign-service": ("python", {"ts-consign-price-service"}),
}


def load(name: str):
    with (GROUND_TRUTH / name).open(encoding="utf-8") as stream:
        return json.load(stream)


def production_files(service: pathlib.Path):
    for path in service.rglob("*"):
        if not path.is_file() or path.suffix not in {".go", ".py"}:
            continue
        if "test" in path.parts or path.name.endswith(("_test.go", "_test.py")):
            continue
        yield path


def fail(message: str):
    print(f"ground-truth error: {message}", file=sys.stderr)
    raise SystemExit(1)


def main():
    services_document = load("services.yaml")
    rest_document = load("rest-edges.yaml")
    endpoint_document = load("endpoint-edges.yaml")
    load("persistence.yaml")
    load("scenarios/selected-services.yaml")

    services = {entry["name"]: entry for entry in services_document["services"]}
    if len(services) != len(services_document["services"]):
        fail("duplicate service identity")

    rest_edges = {tuple(edge) for edge in rest_document["edges"]}
    if len(rest_edges) != len(rest_document["edges"]):
        fail("duplicate service-level REST edge")
    for source, target in rest_edges:
        if source not in services or target not in services:
            fail(f"edge references an unknown service: {source} -> {target}")

    endpoint_keys = set()
    for edge in endpoint_document["edges"]:
        key = (edge["source"], edge["target"], edge["method"], edge["path"])
        if key in endpoint_keys:
            fail(f"duplicate endpoint edge: {key}")
        endpoint_keys.add(key)
        if (edge["source"], edge["target"]) not in rest_edges:
            fail(f"endpoint edge missing service-level edge: {key}")

    for service_name, (language, expected_targets) in REWRITES.items():
        metadata = services.get(service_name)
        if not metadata or metadata.get("language") != language or not metadata.get("rewritten"):
            fail(f"incorrect rewrite metadata for {service_name}")
        service_dir = ROOT / service_name
        java_files = list(service_dir.rglob("*.java"))
        if java_files:
            fail(f"active Java files remain in {service_name}: {java_files[0]}")
        source = "\n".join(path.read_text(encoding="utf-8") for path in production_files(service_dir))
        targets = set(SERVICE_PATTERN.findall(source)) - {service_name}
        if targets != expected_targets:
            fail(f"{service_name} target set is {sorted(targets)}, expected {sorted(expected_targets)}")

    print(f"ground truth valid: {len(services)} services, {len(rest_edges)} REST edges, {len(endpoint_keys)} selected endpoint edges")


if __name__ == "__main__":
    main()
