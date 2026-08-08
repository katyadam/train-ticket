from __future__ import annotations

import httpx

from app.clients.base import Resolver
from app.domain.models import TripInfo, TripResponse

TRAVEL_SERVICE = "ts-travel-service"


class TravelClient:
    def __init__(self, resolver: Resolver, client: httpx.AsyncClient) -> None:
        self.resolver = resolver
        self.client = client

    async def left_trips(self, info: TripInfo) -> list[TripResponse]:
        travel_service_url = await self.resolver.resolve(TRAVEL_SERVICE)
        result = await self.client.post(
            travel_service_url + "/api/v1/travelservice/trips/left",
            json=info.model_dump(),
        )
        result.raise_for_status()
        return [TripResponse.model_validate(item) for item in result.json()["data"]]
