from __future__ import annotations

from contextlib import asynccontextmanager

import httpx
from fastapi import FastAPI, Request
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse, Response
from starlette.middleware.base import BaseHTTPMiddleware

from app.api.routes import create_router
from app.clients.base import Transport
from app.clients.route_plan_client import RoutePlanClient
from app.clients.seat_client import SeatClient
from app.clients.train_client import TrainClient
from app.clients.travel2_client import Travel2Client
from app.clients.travel_client import TravelClient
from app.config import SERVICE_NAME, Config, load_config
from app.discovery.nacos import NacosDiscovery
from app.security.jwt import validate_optional_bearer
from app.service.travel_plan import TravelPlanService


def build_service(discovery: NacosDiscovery, client: httpx.AsyncClient) -> TravelPlanService:
    transport = Transport(discovery, client)
    return TravelPlanService(
        TravelClient(discovery, client),
        Travel2Client(transport),
        RoutePlanClient(transport),
        TrainClient(discovery, client),
        SeatClient(discovery, client),
    )


def create_app(
    service: TravelPlanService | None = None,
    config: Config | None = None,
    discovery: NacosDiscovery | None = None,
    client: httpx.AsyncClient | None = None,
) -> FastAPI:
    config = config or load_config()
    owns_client = client is None
    client = client or httpx.AsyncClient(timeout=None)
    discovery = discovery or NacosDiscovery(
        config.nacos_addrs,
        config.static_targets,
        SERVICE_NAME,
        config.port,
        config.pod_ip,
    )
    service = service or build_service(discovery, client)

    @asynccontextmanager
    async def lifespan(_: FastAPI):
        if config.nacos_enabled:
            await discovery.register()
        try:
            yield
        finally:
            await discovery.close()
            if owns_client:
                await client.aclose()

    app = FastAPI(docs_url=None, redoc_url=None, openapi_url=None, lifespan=lifespan)
    app.add_middleware(CompatibilityMiddleware)
    app.include_router(create_router(service))

    @app.exception_handler(RequestValidationError)
    async def validation_error(_: Request, __: RequestValidationError):
        return Response(status_code=400)

    @app.exception_handler(Exception)
    async def unhandled(_: Request, __: Exception):
        return JSONResponse(status_code=500, content={"status": 500, "error": "Internal Server Error"})

    return app


class CompatibilityMiddleware(BaseHTTPMiddleware):
    async def dispatch(self, request: Request, call_next):
        origin = request.headers.get("origin")
        if request.method == "OPTIONS" and request.headers.get("access-control-request-method"):
            response = Response(status_code=200)
            response.headers["access-control-allow-methods"] = request.headers["access-control-request-method"]
            requested = request.headers.get("access-control-request-headers")
            if requested:
                response.headers["access-control-allow-headers"] = requested
            response.headers["access-control-max-age"] = "3600"
        elif not validate_optional_bearer(request.headers.get("authorization")):
            response = Response(status_code=401)
        else:
            response = await call_next(request)
        if origin:
            response.headers["access-control-allow-origin"] = "*"
            response.headers.append("vary", "Origin")
        response.headers.setdefault("cache-control", "no-cache, no-store, max-age=0, must-revalidate")
        response.headers.setdefault("pragma", "no-cache")
        response.headers.setdefault("expires", "0")
        return response


app = create_app()
