from __future__ import annotations

import re
from datetime import datetime, timedelta
from zoneinfo import ZoneInfo

from app.clients.route_plan_client import RoutePlanClient
from app.clients.seat_client import SeatClient
from app.clients.train_client import TrainClient
from app.clients.travel2_client import Travel2Client
from app.clients.travel_client import TravelClient
from app.domain.models import (
    RoutePlanResultUnit,
    TransferTravelInfo,
    TravelAdvanceResultUnit,
    TripInfo,
    response,
)

SHANGHAI = ZoneInfo("Asia/Shanghai")
DATE_ONLY = re.compile(r"^(\d{1,4})-(\d{1,2})-(\d{1,2})")
DATE_TIME = re.compile(r"^(\d{1,4})-(\d{1,2})-(\d{1,2}) (\d{1,2}):(\d{1,2}):(\d{1,2})")


class TravelPlanService:
    def __init__(
        self,
        travel: TravelClient,
        travel2: Travel2Client,
        route_plan: RoutePlanClient,
        train: TrainClient,
        seat: SeatClient,
    ) -> None:
        self.travel = travel
        self.travel2 = travel2
        self.route_plan = route_plan
        self.train = train
        self.seat = seat

    async def transfer(self, info: TransferTravelInfo) -> dict:
        departure = java_date_string(info.travelDate)
        first_query = TripInfo(
            startPlace=_normalize(info.startStation),
            endPlace=_normalize(info.viaStation),
            departureTime=departure,
        )
        first_high_speed = await self.travel.left_trips(first_query)
        first_normal = await self.travel2.left_trips(first_query)

        second_query = TripInfo(
            startPlace=_normalize(info.viaStation),
            endPlace=_normalize(info.endStation),
            departureTime=departure,
        )
        second_high_speed = await self.travel.left_trips(second_query)
        second_normal = await self.travel2.left_trips(second_query)

        data = {
            "firstSectionResult": [item.model_dump() for item in first_high_speed + first_normal],
            "secondSectionResult": [item.model_dump() for item in second_high_speed + second_normal],
        }
        return response(1, "Success.", data)

    async def cheapest(self, info: TripInfo) -> dict:
        return await self._advanced(info, self.route_plan.cheapest)

    async def quickest(self, info: TripInfo) -> dict:
        return await self._advanced(info, self.route_plan.quickest)

    async def minimum_station(self, info: TripInfo) -> dict:
        return await self._advanced(info, self.route_plan.minimum_stops)

    async def _advanced(self, info: TripInfo, operation) -> dict:
        route_plan_info = {
            "startStation": _normalize(info.startPlace),
            "endStation": _normalize(info.endPlace),
            "travelDate": info.departureTime,
            "num": 5,
        }
        route_units: list[RoutePlanResultUnit] = await operation(route_plan_info)
        if not route_units:
            return response(0, "Cannot Find", None)

        results: list[TravelAdvanceResultUnit] = []
        for unit in route_units:
            result = TravelAdvanceResultUnit(
                tripId=unit.tripId,
                trainTypeId=unit.trainTypeName,
                startStation=unit.startStation,
                endStation=unit.endStation,
                stopStations=unit.stopStations,
                priceForSecondClassSeat=unit.priceForSecondClassSeat,
                priceForFirstClassSeat=unit.priceForFirstClassSeat,
                startTime=unit.startTime,
                endTime=unit.endTime,
            )
            train_type = await self.train.by_name(unit.trainTypeName)
            first = await self.seat.left_tickets(
                _seat_request(info, unit, seat_type=2, total=train_type.confortClass)
            )
            second = await self.seat.left_tickets(
                _seat_request(info, unit, seat_type=3, total=train_type.economyClass)
            )
            result.numberOfRestTicketFirstClass = first
            result.numberOfRestTicketSecondClass = second
            results.append(result)
        return response(1, "Success", results)


def _seat_request(info: TripInfo, unit: RoutePlanResultUnit, seat_type: int, total: int) -> dict:
    return {
        "travelDate": info.departureTime,
        "trainNumber": unit.tripId,
        "startStation": unit.endStation,
        "destStation": unit.startStation,
        "seatType": seat_type,
        "totalNum": total,
        "stations": unit.stopStations,
    }


def _normalize(value: str | None) -> str | None:
    if value is None or value == "":
        return value
    return value.replace(" ", "").lower()


def java_date_string(value: str | None) -> str:
    if value is None:
        raise TypeError("null date")
    matcher = DATE_TIME if len(value) > 10 else DATE_ONLY
    match = matcher.match(value)
    if not match:
        return datetime.fromtimestamp(0, SHANGHAI).strftime("%Y-%m-%d %H:%M:%S")
    numbers = [int(part) for part in match.groups()]
    while len(numbers) < 6:
        numbers.append(0)
    year, month, day, hour, minute, second = numbers
    try:
        normalized_year, zero_based_month = divmod(year * 12 + month - 1, 12)
        parsed = datetime(normalized_year, zero_based_month + 1, 1, tzinfo=SHANGHAI) + timedelta(
            days=day - 1, hours=hour, minutes=minute, seconds=second
        )
    except (OverflowError, ValueError):
        return datetime.fromtimestamp(0, SHANGHAI).strftime("%Y-%m-%d %H:%M:%S")
    return parsed.strftime("%Y-%m-%d %H:%M:%S")
