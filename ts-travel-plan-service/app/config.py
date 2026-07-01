from __future__ import annotations

import os
from dataclasses import dataclass

SERVICE_NAME = "ts-travel-plan-service"
DEFAULT_PORT = 14322
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
    static_targets: dict[str, str]


def load_config() -> Config:
    addresses = os.getenv("NACOS_ADDRS", DEFAULT_NACOS_ADDRS)
    return Config(
        port=int(os.getenv("PORT", str(DEFAULT_PORT))),
        pod_ip=os.getenv("POD_IP"),
        nacos_enabled=os.getenv("NACOS_ENABLED", "true").lower() != "false",
        nacos_addrs=tuple(item.strip() for item in addresses.split(",") if item.strip()),
        static_targets={
            "ts-travel-service": os.getenv("TRAVEL_SERVICE_URL", "").rstrip("/"),
            "ts-travel2-service": os.getenv("TRAVEL2_SERVICE_URL", "").rstrip("/"),
            "ts-route-plan-service": os.getenv("ROUTE_PLAN_SERVICE_URL", "").rstrip("/"),
            "ts-train-service": os.getenv("TRAIN_SERVICE_URL", "").rstrip("/"),
            "ts-seat-service": os.getenv("SEAT_SERVICE_URL", "").rstrip("/"),
        },
    )
