from __future__ import annotations

import math
from decimal import Decimal
from typing import Protocol

import httpx

CONSIGN_PRICE_SERVICE = "ts-consign-price-service"
HOP_BY_HOP = {"connection", "content-length", "host", "transfer-encoding"}


class Resolver(Protocol):
    async def resolve(self, service_name: str) -> str: ...


class ConsignPriceClient:
    def __init__(self, resolver: Resolver, client: httpx.AsyncClient) -> None:
        self.resolver = resolver
        self.client = client

    async def price(self, weight: float, within: bool, inbound_headers: dict[str, str]) -> float:
        base_url = await self.resolver.resolve(CONSIGN_PRICE_SERVICE)
        path = (
            "/api/v1/consignpriceservice/consignprice/"
            + java_double_string(weight)
            + "/"
            + str(within).lower()
        )
        headers = {name: value for name, value in inbound_headers.items() if name.lower() not in HOP_BY_HOP}
        result = await self.client.get(base_url.rstrip("/") + path, headers=headers)
        result.raise_for_status()
        value = result.json()["data"]
        if not isinstance(value, (float, int)) or isinstance(value, bool):
            raise TypeError("consign price is not numeric")
        return float(value)


def java_double_string(value: float) -> str:
    if math.isnan(value):
        return "NaN"
    if math.isinf(value):
        return "Infinity" if value > 0 else "-Infinity"
    if value == 0:
        return "-0.0" if math.copysign(1.0, value) < 0 else "0.0"
    decimal = Decimal(repr(value))
    exponent = decimal.adjusted()
    if exponent < -3 or exponent >= 7:
        mantissa = format(decimal.scaleb(-exponent), "f").rstrip("0").rstrip(".")
        if "." not in mantissa:
            mantissa += ".0"
        return f"{mantissa}E{exponent}"
    return str(value)
