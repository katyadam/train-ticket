from __future__ import annotations

import asyncio
import ipaddress
import json
import socket
from collections import defaultdict
from urllib.parse import urlsplit, urlunsplit

import httpx


class NacosDiscovery:
    def __init__(
        self,
        addresses: tuple[str, ...],
        static_targets: dict[str, str],
        service_name: str,
        port: int,
        pod_ip: str | None = None,
        client: httpx.AsyncClient | None = None,
    ) -> None:
        self.addresses = tuple(_normalize_address(address) for address in addresses)
        self.static_targets = static_targets
        self.service_name = service_name
        self.port = port
        self.pod_ip = pod_ip
        self.client = client or httpx.AsyncClient(timeout=5.0)
        self._owns_client = client is None
        self._next: defaultdict[str, int] = defaultdict(int)
        self._heartbeat_task: asyncio.Task[None] | None = None
        self._registered_address: str | None = None

    async def resolve(self, service_name: str) -> str:
        static = self.static_targets.get(service_name, "")
        if static:
            return static.rstrip("/")
        last_error: Exception | None = None
        for address in self.addresses:
            try:
                result = await self.client.get(
                    f"{address}/v1/ns/instance/list",
                    params={"serviceName": service_name, "healthyOnly": "true"},
                )
                result.raise_for_status()
                hosts = [
                    host
                    for host in result.json().get("hosts", [])
                    if host.get("healthy") and host.get("enabled") and host.get("ip") and host.get("port")
                ]
                if not hosts:
                    raise RuntimeError(f"Nacos returned no healthy instance for {service_name}")
                index = self._next[service_name] % len(hosts)
                self._next[service_name] += 1
                host = hosts[index]
                return f"http://{host['ip']}:{host['port']}"
            except Exception as exc:  # preserve a single discovery failure at the call site
                last_error = exc
        raise RuntimeError(f"cannot resolve {service_name}") from last_error

    async def register(self) -> None:
        if not self.pod_ip:
            self.pod_ip = _detect_pod_ip()
        params = self._instance_params()
        last_error: Exception | None = None
        for address in self.addresses:
            try:
                result = await self.client.post(f"{address}/v1/ns/instance", data=params)
                result.raise_for_status()
                if result.text.strip() != "ok":
                    raise RuntimeError(f"unexpected Nacos registration response: {result.text}")
                self._registered_address = address
                self._heartbeat_task = asyncio.create_task(self._heartbeat())
                return
            except Exception as exc:
                last_error = exc
        raise RuntimeError("cannot register with Nacos") from last_error

    async def close(self) -> None:
        if self._heartbeat_task:
            self._heartbeat_task.cancel()
            try:
                await self._heartbeat_task
            except asyncio.CancelledError:
                pass
        if self._registered_address:
            result = await self.client.delete(
                f"{self._registered_address}/v1/ns/instance",
                params=self._instance_params(),
            )
            result.raise_for_status()
        if self._owns_client:
            await self.client.aclose()

    async def _heartbeat(self) -> None:
        assert self._registered_address is not None
        while True:
            await asyncio.sleep(5)
            beat = {
                "ip": self.pod_ip,
                "port": self.port,
                "serviceName": self.service_name,
                "weight": 1,
                "healthy": True,
                "enabled": True,
                "ephemeral": True,
                "cluster": "DEFAULT",
            }
            await self.client.put(
                f"{self._registered_address}/v1/ns/instance/beat",
                data={**self._instance_params(), "beat": json.dumps(beat, separators=(",", ":"))},
            )

    def _instance_params(self) -> dict[str, str]:
        return {
            "serviceName": self.service_name,
            "ip": str(self.pod_ip),
            "port": str(self.port),
            "clusterName": "DEFAULT",
            "ephemeral": "true",
        }


def _normalize_address(address: str) -> str:
    address = address.strip().rstrip("/")
    if "://" not in address:
        address = "http://" + address
    parts = urlsplit(address)
    hostname = parts.hostname or ""
    port = parts.port or 8848
    if ":" in hostname:
        hostname = f"[{hostname}]"
    path = parts.path.rstrip("/")
    if not path.endswith("/nacos"):
        path += "/nacos"
    return urlunsplit((parts.scheme, f"{hostname}:{port}", path, "", ""))


def _detect_pod_ip() -> str:
    configured = socket.gethostbyname(socket.gethostname())
    if not ipaddress.ip_address(configured).is_loopback:
        return configured
    with socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as sock:
        sock.connect(("8.8.8.8", 80))
        return str(sock.getsockname()[0])
