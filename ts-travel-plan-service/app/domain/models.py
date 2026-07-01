from __future__ import annotations

from typing import Any

from pydantic import BaseModel, ConfigDict


class WireModel(BaseModel):
    model_config = ConfigDict(extra="ignore")


class TripId(WireModel):
    type: str | None = None
    number: str | None = None

    def train_number(self) -> str:
        if self.type is None or self.number is None:
            raise TypeError("invalid TripId")
        return self.type + self.number


class TripResponse(WireModel):
    tripId: TripId | None = None
    trainTypeName: str | None = ""
    startStation: str | None = ""
    terminalStation: str | None = ""
    startTime: str | None = ""
    endTime: str | None = ""
    economyClass: int = 0
    confortClass: int = 0
    priceForEconomyClass: str | None = ""
    priceForConfortClass: str | None = ""


class TripInfo(WireModel):
    startPlace: str | None = ""
    endPlace: str | None = ""
    departureTime: str | None = ""


class TransferTravelInfo(WireModel):
    startStation: str | None = None
    viaStation: str | None = None
    endStation: str | None = None
    travelDate: str | None = None
    trainType: str | None = None


class RoutePlanResultUnit(WireModel):
    tripId: str | None = None
    trainTypeName: str | None = None
    startStation: str | None = None
    endStation: str | None = None
    stopStations: list[str] | None = None
    priceForSecondClassSeat: str | None = None
    priceForFirstClassSeat: str | None = None
    startTime: str | None = None
    endTime: str | None = None


class TrainType(WireModel):
    id: str | None = None
    name: str | None = None
    economyClass: int = 0
    confortClass: int = 0
    averageSpeed: int = 0


class TravelAdvanceResultUnit(WireModel):
    tripId: str | None = None
    trainTypeId: str | None = None
    startStation: str | None = None
    endStation: str | None = None
    stopStations: list[str] | None = None
    priceForSecondClassSeat: str | None = None
    numberOfRestTicketSecondClass: int = 0
    priceForFirstClassSeat: str | None = None
    numberOfRestTicketFirstClass: int = 0
    startTime: str | None = None
    endTime: str | None = None


def response(status: int, msg: str, data: Any) -> dict[str, Any]:
    if isinstance(data, BaseModel):
        data = data.model_dump()
    elif isinstance(data, list):
        data = [item.model_dump() if isinstance(item, BaseModel) else item for item in data]
    return {"status": status, "msg": msg, "data": data}
