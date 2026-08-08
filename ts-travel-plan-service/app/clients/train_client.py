from __future__ import annotations

import httpx

from app.clients.base import Resolver
from app.domain.models import TrainType

TRAIN_SERVICE = "ts-train-service"


class TrainClient:
    def __init__(self, resolver: Resolver, client: httpx.AsyncClient) -> None:
        self.resolver = resolver
        self.client = client

    async def by_name(self, train_type_name: str | None) -> TrainType:
        if train_type_name is None:
            raise TypeError("null trainTypeName")
        train_service_url = await self.resolver.resolve(TRAIN_SERVICE)
        result = await self.client.get(
            train_service_url + "/api/v1/trainservice/trains/byName/" + train_type_name,
        )
        result.raise_for_status()
        return TrainType.model_validate(result.json()["data"])
