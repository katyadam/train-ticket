from __future__ import annotations

from contextlib import asynccontextmanager

import httpx
from fastapi import FastAPI, Request
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse, Response
from starlette.middleware.base import BaseHTTPMiddleware

from app.api.routes import BASE_PATH, create_router
from app.clients.consign_price_client import ConsignPriceClient
from app.config import SERVICE_NAME, Config, load_config
from app.discovery.nacos import NacosDiscovery
from app.repository.consign import ConsignRepository
from app.security.jwt import parse_optional_bearer
from app.service.consign import ConsignService


def create_app(
    service: ConsignService | None = None,
    config: Config | None = None,
    repository: ConsignRepository | None = None,
    discovery: NacosDiscovery | None = None,
    client: httpx.AsyncClient | None = None,
) -> FastAPI:
    config = config or load_config()
    owns_repository = repository is None and service is None
    repository = repository or (ConsignRepository(config.database_url) if service is None else None)
    owns_client = client is None
    client = client or httpx.AsyncClient(timeout=None)
    discovery = discovery or NacosDiscovery(
        config.nacos_addrs,
        {"ts-consign-price-service": config.price_service_url},
        SERVICE_NAME,
        config.port,
        config.pod_ip,
    )
    if service is None:
        assert repository is not None
        service = ConsignService(repository, ConsignPriceClient(discovery, client))

    @asynccontextmanager
    async def lifespan(_: FastAPI):
        if repository is not None:
            repository.ensure_schema()
        if config.nacos_enabled:
            await discovery.register()
        try:
            yield
        finally:
            await discovery.close()
            if owns_client:
                await client.aclose()
            if owns_repository and repository is not None:
                repository.close()

    app = FastAPI(docs_url=None, redoc_url=None, openapi_url=None, lifespan=lifespan)
    app.add_middleware(SecurityAndCorsMiddleware)
    app.include_router(create_router(service))

    @app.exception_handler(RequestValidationError)
    async def validation_error(_: Request, __: RequestValidationError):
        return Response(status_code=400)

    @app.exception_handler(ValueError)
    async def invalid_path(_: Request, __: ValueError):
        return JSONResponse(status_code=500, content={"status": 500, "error": "Internal Server Error"})

    @app.exception_handler(Exception)
    async def unhandled(_: Request, __: Exception):
        return JSONResponse(status_code=500, content={"status": 500, "error": "Internal Server Error"})

    return app


class SecurityAndCorsMiddleware(BaseHTTPMiddleware):
    async def dispatch(self, request: Request, call_next):
        origin = request.headers.get("origin")
        if request.method == "OPTIONS" and request.headers.get("access-control-request-method"):
            response = Response(status_code=200)
            response.headers["access-control-allow-methods"] = request.headers["access-control-request-method"]
            requested = request.headers.get("access-control-request-headers")
            if requested:
                response.headers["access-control-allow-headers"] = requested
            response.headers["access-control-max-age"] = "3600"
        elif request.url.path.startswith(BASE_PATH + "/"):
            try:
                auth = parse_optional_bearer(request.headers.get("authorization"))
            except ValueError:
                response = Response(status_code=401)
            else:
                if not auth.present or not ({"ROLE_ADMIN", "ROLE_USER"} & auth.roles):
                    response = Response(status_code=403)
                else:
                    response = await call_next(request)
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
