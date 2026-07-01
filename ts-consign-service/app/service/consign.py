from __future__ import annotations

import uuid
from typing import Protocol

from app.clients.consign_price_client import ConsignPriceClient, java_double_string
from app.domain.models import Consign, ConsignRecord, response


class Repository(Protocol):
    def find_by_id(self, record_id: str) -> ConsignRecord | None: ...
    def find_by_account_id(self, account_id: str) -> list[ConsignRecord]: ...
    def find_by_order_id(self, order_id: str) -> ConsignRecord | None: ...
    def find_by_consignee(self, consignee: str) -> list[ConsignRecord]: ...
    def save(self, record: ConsignRecord) -> ConsignRecord: ...


class ConsignService:
    def __init__(self, repository: Repository, pricing: ConsignPriceClient) -> None:
        self.repository = repository
        self.pricing = pricing

    async def insert(self, request: Consign, headers: dict[str, str]) -> dict:
        if request.orderId is None or request.accountId is None:
            raise TypeError("orderId and accountId are required by Java toString semantics")
        price = await self.pricing.price(request.weight, request.within, headers)
        record = ConsignRecord(
            id=str(uuid.uuid4()),
            orderId=str(request.orderId),
            accountId=str(request.accountId),
            handleDate=request.handleDate,
            targetDate=request.targetDate,
            from_=request.from_,
            to=request.to,
            consignee=request.consignee,
            phone=request.phone,
            weight=request.weight,
            price=price,
        )
        result = self.repository.save(record)
        return response(1, "You have consigned successfully! The price is " + java_double_string(result.price), result)

    async def update(self, request: Consign, headers: dict[str, str]) -> dict:
        if request.id is None:
            raise TypeError("null consign id")
        original = self.repository.find_by_id(request.id)
        if original is None:
            return await self.insert(request, headers)
        if request.accountId is None:
            raise TypeError("null account id")
        original.accountId = str(request.accountId)
        original.handleDate = request.handleDate
        original.targetDate = request.targetDate
        original.from_ = request.from_
        original.to = request.to
        original.consignee = request.consignee
        original.phone = request.phone
        if original.weight != request.weight:
            original.price = await self.pricing.price(request.weight, request.within, headers)
        original.consignee = request.consignee
        original.phone = request.phone
        original.weight = request.weight
        self.repository.save(original)
        return response(1, "Update consign success", original)

    def by_account(self, account_id: str) -> dict:
        records = self.repository.find_by_account_id(_uuid_string(account_id))
        if records:
            return response(1, "Find consign by account id success", records)
        return response(0, "No Content according to accountId", None)

    def by_order(self, order_id: str) -> dict:
        record = self.repository.find_by_order_id(_uuid_string(order_id))
        if record:
            return response(1, "Find consign by order id success", record)
        return response(0, "No Content according to order id", None)

    def by_consignee(self, consignee: str) -> dict:
        records = self.repository.find_by_consignee(consignee)
        if records:
            return response(1, "Find consign by consignee success", records)
        return response(0, "No Content according to consignee", None)


def _uuid_string(value: str) -> str:
    return str(uuid.UUID(value))
