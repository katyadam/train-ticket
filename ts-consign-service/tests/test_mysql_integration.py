from __future__ import annotations

import os

import pytest
from sqlalchemy import text

from app.domain.models import ConsignRecord
from app.repository.consign import ConsignRepository


@pytest.mark.skipif(not os.getenv("CONSIGN_TEST_MYSQL_URL"), reason="requires a disposable MySQL 5.7 database")
def test_mysql57_schema_round_trip_and_restart():
    repository = ConsignRepository(os.environ["CONSIGN_TEST_MYSQL_URL"])
    try:
        repository.ensure_schema()
        with repository.engine.begin() as connection:
            ddl = connection.execute(text("SHOW CREATE TABLE consign_record")).one()[1]
            expected_fragments = (
                "`consign_record_id` varchar(255) NOT NULL",
                "`user_id` varchar(255) DEFAULT NULL",
                "`consign_record_price` double DEFAULT NULL",
                "`weight` double NOT NULL",
                "ENGINE=MyISAM",
            )
            assert all(fragment in ddl for fragment in expected_fragments), ddl
            assert ddl.index("`user_id`") < ddl.index("`consignee`") < ddl.index("`from_place`")
            assert ddl.index("`consign_record_price`") < ddl.index("`target_date`") < ddl.index("`weight`")
            connection.execute(text("DELETE FROM consign_record"))
        expected = ConsignRecord(
            id="record",
            orderId="11111111-1111-1111-1111-111111111111",
            accountId="22222222-2222-2222-2222-222222222222",
            handleDate="2020-01-01",
            targetDate="2020-01-02",
            from_="shanghai",
            to="beijing",
            consignee="Ada",
            phone="123",
            weight=1.0,
            price=3.0,
        )
        repository.save(expected)
        restarted = ConsignRepository(os.environ["CONSIGN_TEST_MYSQL_URL"])
        try:
            assert restarted.find_by_id("record") == expected
        finally:
            restarted.close()
    finally:
        repository.close()
