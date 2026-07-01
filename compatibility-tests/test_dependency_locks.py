from __future__ import annotations

import re
from pathlib import Path

import pytest


ROOT = Path(__file__).resolve().parents[1]
PYTHON_SERVICES = ("ts-travel-plan-service", "ts-consign-service")
GO_SERVICES = ("ts-route-plan-service", "ts-station-service")


@pytest.mark.parametrize("service", PYTHON_SERVICES)
def test_python_dependencies_are_pinned_and_hash_checked(service: str) -> None:
    directory = ROOT / service
    for lock_name in ("requirements.lock", "build-requirements.lock"):
        lock_text = (directory / lock_name).read_text()
        requirements = re.findall(r"(?m)^([A-Za-z0-9_.-]+)==([^ ;\\]+)", lock_text)
        assert requirements, f"{lock_name} has no pinned requirements"
        assert len(re.findall(r"--hash=sha256:[0-9a-f]{64}", lock_text)) >= len(requirements)

    dockerfile = (directory / "Dockerfile").read_text()
    assert "python:3.11.11-bookworm AS build" in dockerfile
    assert "python:3.11.11-slim-bookworm" in dockerfile
    assert dockerfile.count("--require-hashes") == 2
    assert "wheels" not in dockerfile
    assert "--no-index" not in dockerfile


@pytest.mark.parametrize("service", GO_SERVICES)
def test_go_dependencies_are_checksum_locked(service: str) -> None:
    directory = ROOT / service
    assert (directory / "go.sum").read_text().strip()

    dockerfile = (directory / "Dockerfile").read_text()
    assert "go mod download" in dockerfile
    assert "-mod=readonly" in dockerfile
    assert "COPY vendor" not in dockerfile
