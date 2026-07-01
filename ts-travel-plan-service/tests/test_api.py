from __future__ import annotations

import base64
import hashlib
import hmac
import json
import time

import httpx
import pytest

from app.config import Config
from app.main import create_app


class StubService:
    async def transfer(self, _):
        return {"status": 1, "msg": "Success.", "data": {"firstSectionResult": [], "secondSectionResult": []}}

    async def cheapest(self, _):
        return {"status": 1, "msg": "Success", "data": []}

    quickest = cheapest
    minimum_station = cheapest


class StubDiscovery:
    async def close(self):
        return None


def config_for_tests():
    return Config(port=14322, pod_ip=None, nacos_enabled=False, nacos_addrs=(), static_targets={})


@pytest.mark.asyncio
async def test_public_surface_docs_and_optional_invalid_token_behavior():
    app = create_app(service=StubService(), config=config_for_tests(), discovery=StubDiscovery())
    transport = httpx.ASGITransport(app=app, raise_app_exceptions=False)
    async with httpx.AsyncClient(transport=transport, base_url="http://test") as client:
        welcome = await client.get("/api/v1/travelplanservice/welcome")
        assert welcome.status_code == 200
        assert welcome.text == "Welcome to [ TravelPlan Service ] !"

        invalid = await client.get(
            "/api/v1/travelplanservice/welcome", headers={"Authorization": "Bearer invalid"}
        )
        assert invalid.status_code == 401
        assert invalid.content == b""

        non_bearer = await client.get(
            "/api/v1/travelplanservice/welcome", headers={"Authorization": "Basic ignored"}
        )
        assert non_bearer.status_code == 200

        valid = await client.get(
            "/api/v1/travelplanservice/welcome", headers={"Authorization": "Bearer " + token(time.time() + 60)}
        )
        assert valid.status_code == 200

        assert (await client.get("/docs")).status_code == 404
        assert (await client.get("/redoc")).status_code == 404
        assert (await client.get("/openapi.json")).status_code == 404


@pytest.mark.asyncio
async def test_cors_and_validation_status():
    app = create_app(service=StubService(), config=config_for_tests(), discovery=StubDiscovery())
    transport = httpx.ASGITransport(app=app, raise_app_exceptions=False)
    async with httpx.AsyncClient(transport=transport, base_url="http://test") as client:
        preflight = await client.options(
            "/api/v1/travelplanservice/travelPlan/cheapest",
            headers={
                "Origin": "https://example.test",
                "Access-Control-Request-Method": "POST",
                "Access-Control-Request-Headers": "authorization,content-type",
            },
        )
        assert preflight.status_code == 200
        assert preflight.headers["access-control-allow-origin"] == "*"
        assert preflight.headers["access-control-allow-methods"] == "POST"
        assert preflight.headers["access-control-max-age"] == "3600"

        malformed = await client.post(
            "/api/v1/travelplanservice/travelPlan/cheapest",
            content=b"{",
            headers={"content-type": "application/json"},
        )
        assert malformed.status_code == 400


def token(expiration: float) -> str:
    header = _encode({"alg": "HS256", "typ": "JWT"})
    payload = _encode({"sub": "user", "roles": ["ROLE_USER"], "exp": int(expiration)})
    signature = hmac.new(b"secret", f"{header}.{payload}".encode(), hashlib.sha256).digest()
    return f"{header}.{payload}.{base64.urlsafe_b64encode(signature).rstrip(b'=').decode()}"


def _encode(value: dict) -> str:
    raw = json.dumps(value, separators=(",", ":")).encode()
    return base64.urlsafe_b64encode(raw).rstrip(b"=").decode()
