from __future__ import annotations

from app.clients.base import Transport
from app.domain.models import TrainType

TRAIN_SERVICE = "ts-train-service"


class TrainClient:
    def __init__(self, transport: Transport) -> None:
        self.transport = transport

    async def by_name(self, train_type_name: str | None) -> TrainType:
        if train_type_name is None:
            raise TypeError("null trainTypeName")
        result = await self.transport.request(
            TRAIN_SERVICE,
            "GET",
            "/api/v1/trainservice/trains/byName/" + train_type_name,
        )
        return TrainType.model_validate(result["data"])
