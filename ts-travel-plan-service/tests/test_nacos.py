from __future__ import annotations

import httpx
import pytest

from app.discovery.nacos import NacosDiscovery


@pytest.mark.asyncio
async def test_nacos_registration_resolution_and_deregistration_protocol():
    calls = []

    def handler(request: httpx.Request):
        calls.append((request.method, request.url.path, request.url.params, request.content.decode()))
        if request.method == "POST":
            return httpx.Response(200, text="ok")
        if request.method == "GET":
            return httpx.Response(
                200,
                json={"hosts": [{"ip": "10.0.0.8", "port": 14578, "healthy": True, "enabled": True}]},
            )
        if request.method == "DELETE":
            return httpx.Response(200, text="ok")
        return httpx.Response(404)

    client = httpx.AsyncClient(transport=httpx.MockTransport(handler))
    discovery = NacosDiscovery(
        ("http://nacos:8848",),
        {},
        "ts-travel-plan-service",
        14322,
        "10.0.0.7",
        client,
    )
    await discovery.register()
    assert await discovery.resolve("ts-route-plan-service") == "http://10.0.0.8:14578"
    await discovery.close()
    await client.aclose()

    assert [(method, path) for method, path, _, _ in calls] == [
        ("POST", "/nacos/v1/ns/instance"),
        ("GET", "/nacos/v1/ns/instance/list"),
        ("DELETE", "/nacos/v1/ns/instance"),
    ]
    assert "serviceName=ts-travel-plan-service" in calls[0][3]
    assert "ip=10.0.0.7" in calls[0][3]
