from __future__ import annotations

from typing import Any, Protocol

import httpx


class Resolver(Protocol):
    async def resolve(self, service_name: str) -> str: ...


class Transport:
    def __init__(self, resolver: Resolver, client: httpx.AsyncClient) -> None:
        self.resolver = resolver
        self.client = client

    async def request(
        self,
        service_name: str,
        method: str,
        path: str,
        *,
        json: Any | None = None,
    ) -> dict[str, Any]:
        base_url = await self.resolver.resolve(service_name)
        result = await self.client.request(method, base_url.rstrip("/") + path, json=json)
        result.raise_for_status()
        return result.json()
