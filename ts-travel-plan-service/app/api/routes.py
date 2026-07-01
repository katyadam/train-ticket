from __future__ import annotations

from fastapi import APIRouter
from fastapi.responses import PlainTextResponse

from app.domain.models import TransferTravelInfo, TripInfo
from app.service.travel_plan import TravelPlanService

BASE_PATH = "/api/v1/travelplanservice"


def create_router(service: TravelPlanService) -> APIRouter:
    router = APIRouter(prefix=BASE_PATH)

    @router.get("/welcome", response_class=PlainTextResponse)
    async def welcome() -> str:
        return "Welcome to [ TravelPlan Service ] !"

    @router.post("/travelPlan/transferResult")
    async def transfer(info: TransferTravelInfo):
        return await service.transfer(info)

    @router.post("/travelPlan/cheapest")
    async def cheapest(info: TripInfo):
        return await service.cheapest(info)

    @router.post("/travelPlan/quickest")
    async def quickest(info: TripInfo):
        return await service.quickest(info)

    @router.post("/travelPlan/minStation")
    async def minimum_station(info: TripInfo):
        return await service.minimum_station(info)

    return router
