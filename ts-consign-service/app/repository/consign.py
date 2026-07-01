from __future__ import annotations

from sqlalchemy import Double, String, create_engine, select
from sqlalchemy.orm import DeclarativeBase, Mapped, Session, mapped_column, sessionmaker

from app.domain.models import ConsignRecord


class Base(DeclarativeBase):
    pass


class ConsignRow(Base):
    __tablename__ = "consign_record"
    __table_args__ = {"mysql_engine": "MyISAM"}

    id: Mapped[str] = mapped_column("consign_record_id", String(255), primary_key=True)
    account_id: Mapped[str | None] = mapped_column("user_id", String(255), nullable=True)
    consignee: Mapped[str | None] = mapped_column(String(255), nullable=True)
    from_place: Mapped[str | None] = mapped_column(String(255), nullable=True)
    handle_date: Mapped[str | None] = mapped_column(String(255), nullable=True)
    order_id: Mapped[str | None] = mapped_column(String(255), nullable=True)
    phone: Mapped[str | None] = mapped_column("consign_record_phone", String(255), nullable=True)
    price: Mapped[float | None] = mapped_column("consign_record_price", Double, nullable=True)
    target_date: Mapped[str | None] = mapped_column(String(255), nullable=True)
    to_place: Mapped[str | None] = mapped_column(String(255), nullable=True)
    weight: Mapped[float] = mapped_column(Double, nullable=False)


class ConsignRepository:
    def __init__(self, database_url: str) -> None:
        self.engine = create_engine(database_url, pool_pre_ping=True)
        self.sessions = sessionmaker(self.engine, expire_on_commit=False)

    def ensure_schema(self) -> None:
        Base.metadata.create_all(self.engine)

    def close(self) -> None:
        self.engine.dispose()

    def find_by_id(self, record_id: str) -> ConsignRecord | None:
        with self.sessions() as session:
            row = session.get(ConsignRow, record_id)
            return _record(row) if row else None

    def find_by_account_id(self, account_id: str) -> list[ConsignRecord]:
        with self.sessions() as session:
            rows = session.scalars(select(ConsignRow).where(ConsignRow.account_id == account_id)).all()
            return [_record(row) for row in rows]

    def find_by_order_id(self, order_id: str) -> ConsignRecord | None:
        with self.sessions() as session:
            row = session.scalars(select(ConsignRow).where(ConsignRow.order_id == order_id)).one_or_none()
            return _record(row) if row else None

    def find_by_consignee(self, consignee: str) -> list[ConsignRecord]:
        with self.sessions() as session:
            rows = session.scalars(select(ConsignRow).where(ConsignRow.consignee == consignee)).all()
            return [_record(row) for row in rows]

    def save(self, record: ConsignRecord) -> ConsignRecord:
        with self.sessions.begin() as session:
            row = session.get(ConsignRow, record.id)
            if row is None:
                row = ConsignRow(id=record.id)
                session.add(row)
            _copy_to_row(record, row)
        return record


def _copy_to_row(record: ConsignRecord, row: ConsignRow) -> None:
    row.order_id = record.orderId
    row.account_id = record.accountId
    row.handle_date = record.handleDate
    row.target_date = record.targetDate
    row.from_place = record.from_
    row.to_place = record.to
    row.consignee = record.consignee
    row.phone = record.phone
    row.weight = record.weight
    row.price = record.price


def _record(row: ConsignRow) -> ConsignRecord:
    return ConsignRecord(
        id=row.id,
        orderId=row.order_id,
        accountId=row.account_id,
        handleDate=row.handle_date,
        targetDate=row.target_date,
        from_=row.from_place,
        to=row.to_place,
        consignee=row.consignee,
        phone=row.phone,
        weight=row.weight,
        price=row.price,
    )
