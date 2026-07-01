from __future__ import annotations

from app.clients.base import Transport
from app.domain.models import RoutePlanResultUnit

ROUTE_PLAN_SERVICE = "ts-route-plan-service"


class RoutePlanClient:
    def __init__(self, transport: Transport) -> None:
        self.transport = transport

    async def cheapest(self, body: dict) -> list[RoutePlanResultUnit]:
        return await self._route_plan("/api/v1/routeplanservice/routePlan/cheapestRoute", body)

    async def quickest(self, body: dict) -> list[RoutePlanResultUnit]:
        return await self._route_plan("/api/v1/routeplanservice/routePlan/quickestRoute", body)

    async def minimum_stops(self, body: dict) -> list[RoutePlanResultUnit]:
        return await self._route_plan("/api/v1/routeplanservice/routePlan/minStopStations", body)

    async def _route_plan(self, path: str, body: dict) -> list[RoutePlanResultUnit]:
        result = await self.transport.request(ROUTE_PLAN_SERVICE, "POST", path, json=body)
        return [RoutePlanResultUnit.model_validate(item) for item in result["data"]]
