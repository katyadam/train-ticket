from __future__ import annotations

import json
from dataclasses import dataclass

import httpx
import pytest

from app.clients.base import Transport
from app.clients.route_plan_client import RoutePlanClient
from app.clients.seat_client import SeatClient
from app.clients.train_client import TrainClient
from app.clients.travel2_client import Travel2Client
from app.clients.travel_client import TravelClient
from app.domain.models import TransferTravelInfo, TripInfo
from app.service.travel_plan import TravelPlanService, java_date_string


@dataclass
class StaticResolver:
    async def resolve(self, service_name: str) -> str:
        return "http://" + service_name


def service_with_transport(handler):
    client = httpx.AsyncClient(transport=httpx.MockTransport(handler))
    resolver = StaticResolver()
    transport = Transport(resolver, client)
    service = TravelPlanService(
        TravelClient(resolver, client),
        Travel2Client(transport),
        RoutePlanClient(transport),
        TrainClient(resolver, client),
        SeatClient(resolver, client),
    )
    return service, client


@pytest.mark.asyncio
async def test_transfer_preserves_four_call_sequence_and_ignores_train_type():
    calls = []

    def handler(request: httpx.Request):
        calls.append((request.url.host, request.url.path, json.loads(request.content)))
        marker = f"{request.url.host}-{len(calls)}"
        return httpx.Response(200, json={"status": 1, "msg": "Success", "data": [trip(marker)]})

    service, client = service_with_transport(handler)
    try:
        result = await service.transfer(
            TransferTravelInfo(
                startStation=" Shang Hai ",
                viaStation=" Nan Jing ",
                endStation=" Bei Jing ",
                travelDate="2020-01-02",
                trainType="ignored",
            )
        )
    finally:
        await client.aclose()

    assert [(target, path) for target, path, _ in calls] == [
        ("ts-travel-service", "/api/v1/travelservice/trips/left"),
        ("ts-travel2-service", "/api/v1/travel2service/trips/left"),
        ("ts-travel-service", "/api/v1/travelservice/trips/left"),
        ("ts-travel2-service", "/api/v1/travel2service/trips/left"),
    ]
    assert calls[0][2] == {
        "startPlace": "shanghai",
        "endPlace": "nanjing",
        "departureTime": "2020-01-02 00:00:00",
    }
    assert calls[2][2] == {
        "startPlace": "nanjing",
        "endPlace": "beijing",
        "departureTime": "2020-01-02 00:00:00",
    }
    assert [item["trainTypeName"] for item in result["data"]["firstSectionResult"]] == [
        "ts-travel-service-1",
        "ts-travel2-service-2",
    ]


@pytest.mark.asyncio
@pytest.mark.parametrize(
    ("method_name", "route_path"),
    [
        ("cheapest", "/api/v1/routeplanservice/routePlan/cheapestRoute"),
        ("quickest", "/api/v1/routeplanservice/routePlan/quickestRoute"),
        ("minimum_station", "/api/v1/routeplanservice/routePlan/minStopStations"),
    ],
)
async def test_advanced_search_preserves_route_train_and_two_seat_calls(method_name, route_path):
    calls = []

    def handler(request: httpx.Request):
        body = json.loads(request.content) if request.content else None
        calls.append((request.url.host, request.method, request.url.path, body))
        if request.url.host == "ts-route-plan-service":
            return httpx.Response(200, json={"status": 1, "msg": "Success", "data": [route_unit()]})
        if request.url.host == "ts-train-service":
            return httpx.Response(
                200,
                json={"status": 1, "msg": "Success", "data": {"name": "GaoTieOne", "economyClass": 22, "confortClass": 11, "averageSpeed": 300}},
            )
        if request.url.host == "ts-seat-service":
            seat_count = 7 if body["seatType"] == 2 else 8
            return httpx.Response(200, json={"status": 1, "msg": "Success", "data": seat_count})
        raise AssertionError(request.url)

    service, client = service_with_transport(handler)
    try:
        result = await getattr(service, method_name)(
            TripInfo(startPlace=" Shang Hai ", endPlace=" Bei Jing ", departureTime="2020-01-02")
        )
    finally:
        await client.aclose()

    assert [(target, method, path) for target, method, path, _ in calls] == [
        ("ts-route-plan-service", "POST", route_path),
        ("ts-train-service", "GET", "/api/v1/trainservice/trains/byName/GaoTieOne"),
        ("ts-seat-service", "POST", "/api/v1/seatservice/seats/left_tickets"),
        ("ts-seat-service", "POST", "/api/v1/seatservice/seats/left_tickets"),
    ]
    assert calls[0][3] == {
        "startStation": "shanghai",
        "endStation": "beijing",
        "travelDate": "2020-01-02",
        "num": 5,
    }
    assert calls[2][3] == {
        "travelDate": "2020-01-02",
        "trainNumber": "G1234",
        "startStation": "beijing",
        "destStation": "shanghai",
        "seatType": 2,
        "totalNum": 11,
        "stations": ["shanghai", "nanjing", "beijing"],
    }
    assert calls[3][3]["seatType"] == 3
    assert calls[3][3]["totalNum"] == 22
    assert result == {
        "status": 1,
        "msg": "Success",
        "data": [
            {
                "tripId": "G1234",
                "trainTypeId": "GaoTieOne",
                "startStation": "shanghai",
                "endStation": "beijing",
                "stopStations": ["shanghai", "nanjing", "beijing"],
                "priceForSecondClassSeat": "95.0",
                "numberOfRestTicketSecondClass": 8,
                "priceForFirstClassSeat": "120.0",
                "numberOfRestTicketFirstClass": 7,
                "startTime": "2020-01-02 08:00:00",
                "endTime": "2020-01-02 12:00:00",
            }
        ],
    }

@pytest.mark.asyncio
async def test_empty_route_plan_returns_domain_failure_without_enrichment_calls():
    calls = []

    def handler(request: httpx.Request):
        calls.append(request.url.host)
        return httpx.Response(200, json={"status": 1, "msg": "Success", "data": []})

    service, client = service_with_transport(handler)
    try:
        result = await service.cheapest(TripInfo(startPlace="a", endPlace="b", departureTime="2020-01-02"))
    finally:
        await client.aclose()
    assert result == {"status": 0, "msg": "Cannot Find", "data": None}
    assert calls == ["ts-route-plan-service"]


@pytest.mark.parametrize(
    ("source", "expected"),
    [
        ("2020-01-02", "2020-01-02 00:00:00"),
        ("2020-02-31", "2020-03-02 00:00:00"),
        ("bad", "1970-01-01 08:00:00"),
    ],
)
def test_java_compatible_date_conversion(source, expected):
    assert java_date_string(source) == expected


def route_unit():
    return {
        "tripId": "G1234",
        "trainTypeName": "GaoTieOne",
        "startStation": "shanghai",
        "endStation": "beijing",
        "stopStations": ["shanghai", "nanjing", "beijing"],
        "priceForSecondClassSeat": "95.0",
        "priceForFirstClassSeat": "120.0",
        "startTime": "2020-01-02 08:00:00",
        "endTime": "2020-01-02 12:00:00",
    }


def trip(marker):
    return {
        "tripId": {"type": "G", "number": "1"},
        "trainTypeName": marker,
        "startStation": "a",
        "terminalStation": "b",
        "startTime": "2020-01-02 08:00:00",
        "endTime": "2020-01-02 09:00:00",
        "economyClass": 1,
        "confortClass": 1,
        "priceForEconomyClass": "1.0",
        "priceForConfortClass": "2.0",
    }
