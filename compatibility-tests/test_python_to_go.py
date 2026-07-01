from __future__ import annotations

import json
import base64
import os
import shutil
import socket
import subprocess
import sys
import tempfile
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

import httpx
import pytest

ROOT = Path(__file__).resolve().parents[1]
TRAVEL_PLAN = ROOT / "ts-travel-plan-service"
if str(TRAVEL_PLAN) not in sys.path:
    sys.path.insert(0, str(TRAVEL_PLAN))

from app.clients.base import Transport
from app.clients.route_plan_client import RoutePlanClient
from app.domain.models import TrainType, TripInfo
from app.service.travel_plan import TravelPlanService


class StaticResolver:
    def __init__(self, targets):
        self.targets = targets

    async def resolve(self, service_name):
        return self.targets[service_name]


class RecordingStub:
    def __init__(self, service_name):
        self.service_name = service_name
        self.calls = []
        owner = self

        class Handler(BaseHTTPRequestHandler):
            def do_POST(self):
                body = self.rfile.read(int(self.headers.get("content-length", "0")))
                owner.calls.append(
                    (self.command, self.path, json.loads(body), {key.lower(): value for key, value in self.headers.items()})
                )
                if self.path.endswith("/trips/left") and service_name == "ts-travel-service":
                    self._json(
                        {
                            "status": 1,
                            "msg": "Success",
                            "data": [
                                {
                                    "tripId": {"type": "G", "number": "1234"},
                                    "trainTypeName": "GaoTieOne",
                                    "startStation": "shanghai",
                                    "terminalStation": "beijing",
                                    "startTime": "2020-01-02 08:00:00",
                                    "endTime": "2020-01-02 12:00:00",
                                    "economyClass": 20,
                                    "confortClass": 10,
                                    "priceForEconomyClass": "95.0",
                                    "priceForConfortClass": "120.0",
                                }
                            ],
                        }
                    )
                elif self.path.endswith("/trips/left"):
                    self._json({"status": 1, "msg": "Success", "data": []})
                else:
                    self.send_error(404)

            def do_GET(self):
                owner.calls.append(
                    (self.command, self.path, None, {key.lower(): value for key, value in self.headers.items()})
                )
                if self.path == "/api/v1/travelservice/routes/G1234":
                    self._json(
                        {
                            "status": 1,
                            "msg": "Success",
                            "data": {"id": "route", "stations": ["shanghai", "nanjing", "beijing"]},
                        }
                    )
                else:
                    self.send_error(404)

            def _json(self, value):
                encoded = json.dumps(value, separators=(",", ":")).encode()
                self.send_response(200)
                self.send_header("content-type", "application/json")
                self.send_header("content-length", str(len(encoded)))
                self.end_headers()
                self.wfile.write(encoded)

            def log_message(self, *_):
                return

        self.server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)

    @property
    def url(self):
        return f"http://127.0.0.1:{self.server.server_port}"

    def start(self):
        self.thread.start()

    def close(self):
        self.server.shutdown()
        self.server.server_close()
        self.thread.join(timeout=5)


class UnusedTravelClient:
    async def left_trips(self, _):
        raise AssertionError("travel-plan transfer client was not expected")


class TrainClient:
    async def by_name(self, name):
        assert name == "GaoTieOne"
        return TrainType(name=name, economyClass=20, confortClass=10, averageSpeed=300)


class SeatClient:
    def __init__(self):
        self.calls = []

    async def left_tickets(self, body):
        self.calls.append(body)
        return 7 if body["seatType"] == 2 else 8


@pytest.mark.asyncio
@pytest.mark.skipif(shutil.which("go") is None, reason="Go toolchain is required")
async def test_python_travel_plan_calls_real_go_route_plan_without_edge_drift():
    travel = RecordingStub("ts-travel-service")
    travel2 = RecordingStub("ts-travel2-service")
    route = RecordingStub("ts-route-service")
    for stub in (travel, travel2, route):
        stub.start()

    with tempfile.TemporaryDirectory() as temporary:
        binary = Path(temporary) / "route-plan"
        environment = {
            **os.environ,
            "GOCACHE": os.environ.get("GOCACHE", str(Path(temporary) / "go-cache")),
        }
        subprocess.run(
            ["go", "build", "-o", str(binary), "./cmd/server"],
            cwd=ROOT / "ts-route-plan-service",
            env=environment,
            check=True,
        )
        port = free_port()
        process = subprocess.Popen(
            [str(binary)],
            env={
                **environment,
                "PORT": str(port),
                "NACOS_ENABLED": "false",
                "TRAVEL_SERVICE_URL": travel.url,
                "TRAVEL2_SERVICE_URL": travel2.url,
                "ROUTE_SERVICE_URL": route.url,
                "SW_AGENT_COLLECTOR_BACKEND_SERVICES": "127.0.0.1:1",
            },
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
        )
        try:
            wait_for_port(port, process)
            client = httpx.AsyncClient(timeout=5)
            route_plan = RoutePlanClient(
                Transport(StaticResolver({"ts-route-plan-service": f"http://127.0.0.1:{port}"}), client)
            )
            seat = SeatClient()
            service = TravelPlanService(UnusedTravelClient(), UnusedTravelClient(), route_plan, TrainClient(), seat)
            result = await service.cheapest(
                TripInfo(startPlace=" Shang Hai ", endPlace=" Bei Jing ", departureTime="2020-01-02")
            )
            await client.aclose()
        finally:
            process.terminate()
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait(timeout=5)
            for stub in (travel, travel2, route):
                stub.close()

    assert result["status"] == 1
    assert result["data"][0]["tripId"] == "G1234"
    assert result["data"][0]["numberOfRestTicketFirstClass"] == 7
    assert result["data"][0]["numberOfRestTicketSecondClass"] == 8
    assert [call[:2] for call in travel.calls] == [
        ("POST", "/api/v1/travelservice/trips/left"),
        ("GET", "/api/v1/travelservice/routes/G1234"),
    ]
    assert [call[:2] for call in travel2.calls] == [
        ("POST", "/api/v1/travel2service/trips/left")
    ]
    assert route.calls == []
    assert seat.calls[0]["startStation"] == "beijing"
    assert seat.calls[0]["destStation"] == "shanghai"
    sw8_headers = [travel.calls[0][3]["sw8"], travel.calls[1][3]["sw8"], travel2.calls[0][3]["sw8"]]
    assert all(header.startswith("1-") for header in sw8_headers)
    assert len({header.split("-")[1] for header in sw8_headers}) == 1
    assert all(_decode_sw8(header.split("-")[4]) == "ts-route-plan-service" for header in sw8_headers)


def free_port():
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        return sock.getsockname()[1]


def wait_for_port(port, process):
    deadline = time.monotonic() + 10
    while time.monotonic() < deadline:
        if process.poll() is not None:
            output = process.stdout.read() if process.stdout else ""
            raise AssertionError(f"route-plan exited early: {output}")
        try:
            with socket.create_connection(("127.0.0.1", port), timeout=0.1):
                return
        except OSError:
            time.sleep(0.05)
    raise AssertionError("route-plan did not listen in time")


def _decode_sw8(value):
    return base64.b64decode(value + "=" * (-len(value) % 4)).decode()
