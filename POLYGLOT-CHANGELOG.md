# Polyglot benchmark change log

## Semantics-preserving rewrite

- Replaced `ts-route-plan-service` and `ts-station-service` with Go implementations.
- Replaced `ts-travel-plan-service` and `ts-consign-service` with Python/FastAPI implementations.
- Removed the four Java modules from the Maven reactor and active analysis corpus.
- Kept the original Nacos names, ports, gateway paths, HTTP methods, DTO spellings, response envelope, MySQL ownership, JWT key/roles, CORS policy, and business REST topology.
- Added explicit target-specific clients and a 45-service/92-edge machine-readable oracle for Voyantclair.
- Added ordered endpoint-call tests, auth and legacy-quirk tests, MySQL 5.7 integration gates, and a real Python travel-plan to Go route-plan HTTP test.
- Captured Hibernate's exact MySQL 5.7 DDL and retained station nullability/key naming plus consign's legacy MyISAM, nullable-price, and column-order schema so Java-image rollback remains data-compatible.
- Added a polyglot top-level build, locked Python dependency graphs, Go checksums, multi-stage images, Compose static-discovery configuration, Kubernetes pod-IP registration, and language-appropriate SkyWalking instrumentation.
- Updated the build-only JaCoCo plugin from 0.8.2 to 0.8.12 so opt-in Java test forks can start under the documented modern JDK instead of aborting before test discovery; unrelated legacy `ts-contacts-service` matcher failures remain outside the polyglot acceptance gate.
- Pinned Go dependencies with `go.sum` and Python dependencies with hash-locked requirement files while excluding generated vendor trees and binary wheel caches from the analysis corpus; Python 3.11 is pinned because SkyWalking Python 1.2.0 skips plugin execution under 3.12.
- Proved SW8 continuity across Python travel-plan → Go route-plan → Java-target stubs and Java-shaped preserve → Python consign → Java pricing, retaining one trace ID and the correct service identity at each language boundary.

The Java reference remains available at commit `313886e99befb94be6cd45f085c98e0019f59829`. No behavior cleanup is included in this rewrite.
