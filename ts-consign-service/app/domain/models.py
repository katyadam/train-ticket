from __future__ import annotations

from typing import Any

from pydantic import BaseModel, ConfigDict, Field


class WireModel(BaseModel):
    model_config = ConfigDict(extra="ignore")


class Consign(WireModel):
    id: str | None = None
    orderId: str | None = None
    accountId: str | None = None
    handleDate: str | None = None
    targetDate: str | None = None
    from_: str | None = Field(default=None, alias="from")
    to: str | None = None
    consignee: str | None = None
    phone: str | None = None
    weight: float = 0.0
    within: bool = False

    model_config = ConfigDict(extra="ignore", populate_by_name=True)

class ConsignRecord(WireModel):
    id: str
    orderId: str | None = None
    accountId: str | None = None
    handleDate: str | None = None
    targetDate: str | None = None
    from_: str | None = Field(default=None, alias="from")
    to: str | None = None
    consignee: str | None = None
    phone: str | None = None
    weight: float = 0.0
    price: float = 0.0

    model_config = ConfigDict(extra="ignore", populate_by_name=True)

    def wire(self) -> dict[str, Any]:
        return {
            "id": self.id,
            "orderId": self.orderId,
            "accountId": self.accountId,
            "handleDate": self.handleDate,
            "targetDate": self.targetDate,
            "from": self.from_,
            "to": self.to,
            "consignee": self.consignee,
            "phone": self.phone,
            "weight": self.weight,
            "price": self.price,
        }


def response(status: int, msg: str, data: Any) -> dict[str, Any]:
    if isinstance(data, ConsignRecord):
        data = data.wire()
    elif isinstance(data, list):
        data = [item.wire() if isinstance(item, ConsignRecord) else item for item in data]
    return {"status": status, "msg": msg, "data": data}
