from __future__ import annotations

import httpx

from app.clients.base import Resolver

SEAT_SERVICE = "ts-seat-service"


class SeatClient:
    def __init__(self, resolver: Resolver, client: httpx.AsyncClient) -> None:
        self.resolver = resolver
        self.client = client

    async def left_tickets(self, body: dict) -> int:
        seat_service_url = await self.resolver.resolve(SEAT_SERVICE)
        result = await self.client.post(
            seat_service_url + "/api/v1/seatservice/seats/left_tickets",
            json=body,
        )
        result.raise_for_status()
        value = result.json()["data"]
        if not isinstance(value, int) or isinstance(value, bool):
            raise TypeError("seat count is not an integer")
        return value
