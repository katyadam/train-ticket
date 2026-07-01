from __future__ import annotations

from fastapi import APIRouter, Request
from fastapi.responses import PlainTextResponse

from app.domain.models import Consign
from app.service.consign import ConsignService

BASE_PATH = "/api/v1/consignservice"


def create_router(service: ConsignService) -> APIRouter:
    router = APIRouter(prefix=BASE_PATH)

    @router.get("/welcome", response_class=PlainTextResponse)
    async def welcome() -> str:
        return "Welcome to [ Consign Service ] !"

    @router.post("/consigns")
    async def insert(consign: Consign, request: Request):
        return await service.insert(consign, dict(request.headers))

    @router.put("/consigns")
    async def update(consign: Consign, request: Request):
        return await service.update(consign, dict(request.headers))

    @router.get("/consigns/account/{record_id}")
    async def by_account(record_id: str):
        return service.by_account(record_id)

    @router.get("/consigns/order/{record_id}")
    async def by_order(record_id: str):
        return service.by_order(record_id)

    @router.get("/consigns/{consignee}")
    async def by_consignee(consignee: str):
        return service.by_consignee(consignee)

    return router
