# Polyglot verification record

Verification date: 2026-07-01 (Europe/Prague)

Reference source: `master` at `313886e99befb94be6cd45f085c98e0019f59829`.

## Completed gates

- The 39 remaining Java Maven modules built successfully under JDK 17 using the benchmark's historical `maven.test.skip` packaging policy.
- Both Go suites and `go vet` passed; the route-plan/station production images built with checksum-verified modules and read-only module graphs.
- Both Python suites passed, including MySQL-gated consign tests; the travel-plan/consign production images built from fully pinned, hash-locked requirements.
- The real Python travel-plan → Go route-plan compatibility test passed with ordered-call, DTO, seat-swap, edge-set, and Go `sw8` assertions.
- The architecture oracle validated 45 services, 92 service-level REST edges, and 32 selected endpoint edges.
- Root, quickstart, and Jaeger Compose profiles rendered; quickstart, tracing, PVC/NFS, Istio, and Jaeger Kubernetes YAML parsed.
- MySQL 5.7 station and consign tests passed against disposable databases. Hibernate's Java-baseline `SHOW CREATE TABLE` output was captured and the replacement DDL was checked for nullability, key name, storage engine, and column order.
- All four production images started as non-root containers. Public welcome routes returned their exact text, protected consign welcome rejected no-token access and accepted a Java-compatible ROLE_USER token, station returned all 13 seeded rows, and FastAPI documentation routes remained absent.

## Trace-continuity evidence

The tracing profile was exercised with the exact container launchers and an unavailable collector, proving that tracing failure does not stop application traffic.

1. `ts-travel-plan-service` → `ts-route-plan-service` → travel stubs:
   - Python exit carrier service: `ts-travel-plan-service`.
   - Go exit carrier service: `ts-route-plan-service`.
   - The Python-to-Go and Go-to-stub carriers had the same trace ID.
2. Java-shaped preserve carrier → `ts-consign-service` → consign-price stub:
   - Inbound trace ID: `java-trace-id`.
   - Outbound trace ID: `java-trace-id`.
   - Outbound carrier service: `ts-consign-service`.
   - Authorization was forwarded; price `5.0` was returned and the record persisted in MySQL 5.7.

Python containers are pinned to 3.11.11 because SkyWalking Python 1.2.0's plugin loader skips plugin execution under Python 3.12. The live test confirmed FastAPI entry and HTTPX exit instrumentation under 3.11.

## Locally verified production images

These are local content IDs, not registry digests:

| Image | Platform | Content ID | Size (bytes) |
|---|---|---|---:|
| `ts-route-plan-service` | linux/arm64 | `sha256:1cb4f9f3146cb2ce993cc651d3c73411ba3616fa24f7739a06251c002459d43d` | 8,089,669 |
| `ts-station-service` | linux/arm64 | `sha256:c604fd50fd7b0c1e833f101c4d193e0c38c50bacfedf7e239cbbdcee00395352` | 8,290,217 |
| `ts-travel-plan-service` | linux/arm64 | `sha256:9badd10b402aca064ff34e41aa9be985c57dfc7dddf09f666d100ca1ae25cee0` | 72,052,879 |
| `ts-consign-service` | linux/arm64 | `sha256:ab69fa5b91df2c63901a18f87b172ed590b7c29aaa789e3eaa442d7484c28dab` | 79,312,922 |

The original Docker Hub base tag `java:8-jre` no longer exists, so no substitute image is mislabeled as the historical Java image. The four built reference-jar hashes are recorded in `baseline/artifacts.json`.

## Baseline limitation retained

An opt-in full test run for the untouched Java reactor is not a green acceptance gate. After upgrading JaCoCo so JDK 17 can start the forks, the run reaches pre-existing `ts-contacts-service` tests and fails because those tests call `toString()` on Mockito's null-returning `any(UUID.class)` matcher. The polyglot acceptance path therefore preserves the repository's original Java packaging policy and does not claim that the unrelated legacy Java suite passes. The Maven reactor must also run on JDK 17: its legacy Lombok dependency does not generate required accessors under JDK 25. All replacement-service, persistence, architecture, deployment, and cross-language gates listed above are green.
