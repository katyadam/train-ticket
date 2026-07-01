from __future__ import annotations

import os
from dataclasses import dataclass
from urllib.parse import quote_plus

SERVICE_NAME = "ts-consign-service"
DEFAULT_PORT = 16111
DEFAULT_NACOS_ADDRS = (
    "nacos-0.nacos-headless.default.svc.cluster.local,"
    "nacos-1.nacos-headless.default.svc.cluster.local,"
    "nacos-2.nacos-headless.default.svc.cluster.local"
)


@dataclass(frozen=True)
class Config:
    port: int
    pod_ip: str | None
    nacos_enabled: bool
    nacos_addrs: tuple[str, ...]
    price_service_url: str
    database_url: str


def load_config() -> Config:
    addresses = os.getenv("NACOS_ADDRS", DEFAULT_NACOS_ADDRS)
    return Config(
        port=int(os.getenv("PORT", str(DEFAULT_PORT))),
        pod_ip=os.getenv("POD_IP"),
        nacos_enabled=os.getenv("NACOS_ENABLED", "true").lower() != "false",
        nacos_addrs=tuple(item.strip() for item in addresses.split(",") if item.strip()),
        price_service_url=os.getenv("CONSIGN_PRICE_SERVICE_URL", "").rstrip("/"),
        database_url=os.getenv("CONSIGN_MYSQL_URL", _database_url()),
    )


def _database_url() -> str:
    user = quote_plus(os.getenv("CONSIGN_MYSQL_USER", "root"))
    password = quote_plus(os.getenv("CONSIGN_MYSQL_PASSWORD", "root"))
    host = os.getenv("CONSIGN_MYSQL_HOST", "ts-consign-mysql")
    port = os.getenv("CONSIGN_MYSQL_PORT", "3306")
    database = os.getenv("CONSIGN_MYSQL_DATABASE", "ts-consign-mysql")
    return f"mysql+pymysql://{user}:{password}@{host}:{port}/{database}?charset=utf8mb4"
