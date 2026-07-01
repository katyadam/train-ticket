from __future__ import annotations

from app.clients.base import Transport
from app.domain.models import TripInfo, TripResponse

TRAVEL_SERVICE = "ts-travel-service"


class TravelClient:
    def __init__(self, transport: Transport) -> None:
        self.transport = transport

    async def left_trips(self, info: TripInfo) -> list[TripResponse]:
        result = await self.transport.request(
            TRAVEL_SERVICE,
            "POST",
            "/api/v1/travelservice/trips/left",
            json=info.model_dump(),
        )
        return [TripResponse.model_validate(item) for item in result["data"]]
