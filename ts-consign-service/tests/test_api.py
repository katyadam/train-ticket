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
    def __init__(self):
        self.last_request = None

    async def insert(self, consign, _headers):
        self.last_request = consign
        return {"status": 1, "msg": "ok", "data": None}

    async def update(self, consign, _headers):
        self.last_request = consign
        return {"status": 1, "msg": "ok", "data": None}

    def by_account(self, _):
        return {"status": 0, "msg": "No Content according to accountId", "data": None}

    def by_order(self, _):
        return {"status": 0, "msg": "No Content according to order id", "data": None}

    def by_consignee(self, _):
        return {"status": 0, "msg": "No Content according to consignee", "data": None}


class StubDiscovery:
    async def close(self):
        return None


def config_for_tests():
    return Config(
        port=16111,
        pod_ip=None,
        nacos_enabled=False,
        nacos_addrs=(),
        price_service_url="",
        database_url="mysql+pymysql://unused",
    )


@pytest.mark.asyncio
async def test_all_routes_require_user_or_admin_and_invalid_tokens_are_401():
    service = StubService()
    app = create_app(service=service, config=config_for_tests(), discovery=StubDiscovery())
    transport = httpx.ASGITransport(app=app, raise_app_exceptions=False)
    async with httpx.AsyncClient(transport=transport, base_url="http://test") as client:
        assert (await client.get("/api/v1/consignservice/welcome")).status_code == 403
        assert (
            await client.get(
                "/api/v1/consignservice/welcome", headers={"Authorization": "Bearer invalid"}
            )
        ).status_code == 401
        assert (
            await client.get(
                "/api/v1/consignservice/welcome", headers={"Authorization": "Bearer " + token(["ROLE_OTHER"])}
            )
        ).status_code == 403
        for role in ("ROLE_USER", "ROLE_ADMIN"):
            response = await client.get(
                "/api/v1/consignservice/welcome", headers={"Authorization": "Bearer " + token([role])}
            )
            assert response.status_code == 200
            assert response.text == "Welcome to [ Consign Service ] !"


@pytest.mark.asyncio
async def test_preserve_wire_contract_uses_within_and_from_properties():
    service = StubService()
    app = create_app(service=service, config=config_for_tests(), discovery=StubDiscovery())
    transport = httpx.ASGITransport(app=app, raise_app_exceptions=False)
    body = {
        "orderId": "11111111-1111-1111-1111-111111111111",
        "accountId": "22222222-2222-2222-2222-222222222222",
        "from": "shanghai",
        "to": "beijing",
        "weight": 1.0,
        "within": True,
    }
    async with httpx.AsyncClient(transport=transport, base_url="http://test") as client:
        response = await client.post(
            "/api/v1/consignservice/consigns",
            json=body,
            headers={"Authorization": "Bearer " + token(["ROLE_USER"])},
        )
        assert response.status_code == 200
        assert service.last_request.from_ == "shanghai"
        assert service.last_request.within is True
        assert (await client.get("/docs")).status_code == 404
        assert (await client.get("/openapi.json")).status_code == 404


@pytest.mark.asyncio
async def test_cors_preflight_and_malformed_body_status():
    app = create_app(service=StubService(), config=config_for_tests(), discovery=StubDiscovery())
    transport = httpx.ASGITransport(app=app, raise_app_exceptions=False)
    async with httpx.AsyncClient(transport=transport, base_url="http://test") as client:
        preflight = await client.options(
            "/api/v1/consignservice/consigns",
            headers={"Origin": "https://example.test", "Access-Control-Request-Method": "POST"},
        )
        assert preflight.status_code == 200
        assert preflight.headers["access-control-allow-origin"] == "*"
        malformed = await client.post(
            "/api/v1/consignservice/consigns",
            content=b"{",
            headers={"Authorization": "Bearer " + token(["ROLE_USER"]), "content-type": "application/json"},
        )
        assert malformed.status_code == 400


def token(roles):
    header = encode({"alg": "HS256", "typ": "JWT"})
    payload = encode({"sub": "user", "roles": roles, "exp": int(time.time() + 60)})
    signature = hmac.new(b"secret", f"{header}.{payload}".encode(), hashlib.sha256).digest()
    return f"{header}.{payload}.{base64.urlsafe_b64encode(signature).rstrip(b'=').decode()}"


def encode(value):
    return base64.urlsafe_b64encode(json.dumps(value, separators=(",", ":")).encode()).rstrip(b"=").decode()
