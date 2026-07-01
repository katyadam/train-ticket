from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SELECTED = {
    "ts-consign-service": 16111,
    "ts-route-plan-service": 14578,
    "ts-station-service": 12345,
    "ts-travel-plan-service": 14322,
}


def deployment(text: str, name: str) -> str:
    pattern = rf"(?ms)^apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: {re.escape(name)}\n.*?(?=^---$|\Z)"
    match = re.search(pattern, text)
    assert match, f"deployment {name} is missing"
    return match.group(0)


def test_selected_services_are_absent_from_maven_and_have_no_active_java():
    root_pom = (ROOT / "pom.xml").read_text()
    for name in SELECTED:
        assert f"<module>{name}</module>" not in root_pom
        assert not list((ROOT / name).rglob("*.java"))
        dockerfile = (ROOT / name / "Dockerfile").read_text()
        assert "-jar" not in dockerfile


def test_kubernetes_service_names_selectors_and_ports_are_unchanged():
    services = (ROOT / "deployment/kubernetes-manifests/quickstart-k8s/yamls/svc.yaml").read_text()
    for name, port in SELECTED.items():
        assert f"name: {name}" in services
        assert f"app: {name}" in services
        assert f"port: {port}" in services


def test_standard_and_tracing_deployments_register_actual_pod_ip():
    standard = (ROOT / "deployment/kubernetes-manifests/quickstart-k8s/yamls/deploy.yaml.sample").read_text()
    tracing = (ROOT / "deployment/kubernetes-manifests/quickstart-k8s/yamls/sw_deploy.yaml.sample").read_text()
    for name, port in SELECTED.items():
        for document in (standard, tracing):
            block = deployment(document, name)
            assert "name: POD_IP" in block
            assert "fieldPath: status.podIP" in block
            assert f"containerPort: {port}" in block
        tracing_block = deployment(tracing, name)
        assert "skywalking-java-agent" not in tracing_block
        assert "JAVA_TOOL_OPTIONS" not in tracing_block
        assert "SW_AGENT_COLLECTOR_BACKEND_SERVICES" in tracing_block
    assert "/opt/venv/bin/sw-python" in deployment(tracing, "ts-consign-service")
    assert "/opt/venv/bin/sw-python" in deployment(tracing, "ts-travel-plan-service")


def test_gateway_route_families_and_targets_are_unchanged():
    gateway = (ROOT / "ts-gateway-service/src/main/resources/application.yml").read_text()
    expected = {
        "ts-consign-service": "/api/v1/consignservice/**",
        "ts-route-plan-service": "/api/v1/routeplanservice/**",
        "ts-station-service": "/api/v1/stationservice/**",
        "ts-travel-plan-service": "/api/v1/travelplanservice/**",
    }
    for name, path in expected.items():
        assert f":{name}}}" in gateway
        assert f"Path={path}" in gateway


def test_compose_uses_static_discovery_without_new_business_targets():
    for compose in (
        ROOT / "docker-compose.yml",
        ROOT / "deployment/docker-compose-manifests/quickstart-docker-compose.yml",
    ):
        text = compose.read_text()
        assert "NACOS_ENABLED: \"false\"" in text
        assert "STATION_MYSQL_HOST: ts-station-mysql" in text
        assert "CONSIGN_MYSQL_HOST: ts-consign-mysql" in text
        assert "CONSIGN_PRICE_SERVICE_URL: http://ts-consign-price-service:16110" in text
        assert "ROUTE_PLAN_SERVICE_URL: http://ts-route-plan-service:14578" in text
