from __future__ import annotations

from app.clients.base import Transport

SEAT_SERVICE = "ts-seat-service"


class SeatClient:
    def __init__(self, transport: Transport) -> None:
        self.transport = transport

    async def left_tickets(self, body: dict) -> int:
        result = await self.transport.request(
            SEAT_SERVICE,
            "POST",
            "/api/v1/seatservice/seats/left_tickets",
            json=body,
        )
        value = result["data"]
        if not isinstance(value, int) or isinstance(value, bool):
            raise TypeError("seat count is not an integer")
        return value
