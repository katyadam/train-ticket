from __future__ import annotations

import copy

import httpx
import pytest

from app.clients.consign_price_client import ConsignPriceClient, java_double_string
from app.domain.models import Consign, ConsignRecord
from app.service.consign import ConsignService


class MemoryRepository:
    def __init__(self):
        self.records: dict[str, ConsignRecord] = {}
        self.saved: list[ConsignRecord] = []

    def find_by_id(self, record_id):
        record = self.records.get(record_id)
        return copy.deepcopy(record) if record else None

    def find_by_account_id(self, account_id):
        return [copy.deepcopy(record) for record in self.records.values() if record.accountId == account_id]

    def find_by_order_id(self, order_id):
        matches = [record for record in self.records.values() if record.orderId == order_id]
        if len(matches) > 1:
            raise RuntimeError("incorrect result size")
        return copy.deepcopy(matches[0]) if matches else None

    def find_by_consignee(self, consignee):
        return [copy.deepcopy(record) for record in self.records.values() if record.consignee == consignee]

    def save(self, record):
        copy_record = copy.deepcopy(record)
        self.records[record.id] = copy_record
        self.saved.append(copy_record)
        return copy_record


class StaticResolver:
    async def resolve(self, service_name):
        assert service_name == "ts-consign-price-service"
        return "http://ts-consign-price-service"


def service_with_price(handler):
    repository = MemoryRepository()
    client = httpx.AsyncClient(transport=httpx.MockTransport(handler))
    return ConsignService(repository, ConsignPriceClient(StaticResolver(), client)), repository, client


def request(**overrides):
    values = {
        "id": "submitted-id",
        "orderId": "11111111-1111-1111-1111-111111111111",
        "accountId": "22222222-2222-2222-2222-222222222222",
        "handleDate": "2020-01-01",
        "targetDate": "2020-01-02",
        "from": "shanghai",
        "to": "beijing",
        "consignee": "Ada",
        "phone": "123",
        "weight": 1.0,
        "within": True,
    }
    values.update(overrides)
    return Consign.model_validate(values)


@pytest.mark.asyncio
async def test_insert_ignores_submitted_id_forwards_headers_prices_then_saves():
    seen = []

    def price(request: httpx.Request):
        seen.append(request)
        return httpx.Response(200, json={"status": 1, "msg": None, "data": 3.0})

    service, repository, client = service_with_price(price)
    try:
        result = await service.insert(
            request(),
            {"authorization": "Bearer token", "x-trace-id": "trace", "host": "original", "content-length": "10"},
        )
    finally:
        await client.aclose()

    assert len(seen) == 1
    assert seen[0].url.path == "/api/v1/consignpriceservice/consignprice/1.0/true"
    assert seen[0].headers["authorization"] == "Bearer token"
    assert seen[0].headers["x-trace-id"] == "trace"
    assert seen[0].headers["host"] == "ts-consign-price-service"
    assert result["status"] == 1
    assert result["msg"] == "You have consigned successfully! The price is 3.0"
    assert result["data"]["id"] != "submitted-id"
    assert result["data"]["from"] == "shanghai"
    assert "within" not in result["data"]
    assert repository.saved[0].price == 3.0


@pytest.mark.asyncio
async def test_missing_id_update_runs_insert_with_a_new_id():
    calls = 0

    def price(_: httpx.Request):
        nonlocal calls
        calls += 1
        return httpx.Response(200, json={"status": 1, "msg": None, "data": 4.0})

    service, repository, client = service_with_price(price)
    try:
        result = await service.update(request(id="absent"), {})
    finally:
        await client.aclose()
    assert calls == 1
    assert result["data"]["id"] != "absent"
    assert len(repository.records) == 1


@pytest.mark.asyncio
async def test_existing_update_keeps_order_id_and_does_not_reprice_equal_weight_or_within_only():
    def unexpected(_: httpx.Request):
        raise AssertionError("pricing must not be called")

    service, repository, client = service_with_price(unexpected)
    repository.save(
        ConsignRecord(
            id="record",
            orderId="old-order",
            accountId="old-account",
            consignee="Old",
            weight=1.0,
            price=9.0,
        )
    )
    try:
        result = await service.update(request(id="record", orderId="new-order", within=False), {})
    finally:
        await client.aclose()
    assert result["msg"] == "Update consign success"
    assert result["data"]["orderId"] == "old-order"
    assert result["data"]["accountId"] == "22222222-2222-2222-2222-222222222222"
    assert result["data"]["price"] == 9.0
    assert "within" not in result["data"]


@pytest.mark.asyncio
async def test_changed_weight_reprices_before_save():
    paths = []

    def price(request: httpx.Request):
        paths.append(request.url.path)
        return httpx.Response(200, json={"status": 1, "msg": None, "data": 12.5})

    service, repository, client = service_with_price(price)
    repository.save(ConsignRecord(id="record", orderId="old", accountId="account", weight=1.0, price=2.0))
    try:
        result = await service.update(request(id="record", weight=2.0, within=False), {})
    finally:
        await client.aclose()
    assert paths == ["/api/v1/consignpriceservice/consignprice/2.0/false"]
    assert result["data"]["price"] == 12.5
    assert repository.records["record"].weight == 2.0


@pytest.mark.asyncio
async def test_query_messages_uuid_validation_and_order_duplicate_failure():
    service, repository, client = service_with_price(lambda _: None)
    account = "22222222-2222-2222-2222-222222222222"
    order = "11111111-1111-1111-1111-111111111111"
    repository.save(ConsignRecord(id="a", orderId=order, accountId=account, consignee="Ada"))
    assert service.by_account(account)["msg"] == "Find consign by account id success"
    assert service.by_order(order)["msg"] == "Find consign by order id success"
    assert service.by_consignee("Ada")["msg"] == "Find consign by consignee success"
    assert service.by_consignee("missing") == {"status": 0, "msg": "No Content according to consignee", "data": None}
    with pytest.raises(ValueError):
        service.by_account("not-a-uuid")
    repository.save(ConsignRecord(id="b", orderId=order, accountId=account))
    with pytest.raises(RuntimeError):
        service.by_order(order)
    await client.aclose()


@pytest.mark.parametrize(
    ("value", "expected"),
    [
        (1.0, "1.0"),
        (-0.0, "-0.0"),
        (0.0001, "1.0E-4"),
        (9999999.0, "9999999.0"),
        (10000000.0, "1.0E7"),
        (1.23456789012345e20, "1.23456789012345E20"),
        (float("inf"), "Infinity"),
        (float("-inf"), "-Infinity"),
    ],
)
def test_java_double_path_format(value, expected):
    assert java_double_string(value) == expected
