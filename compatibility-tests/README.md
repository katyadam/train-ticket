# Compatibility tests

This directory contains language-neutral and cross-language gates. Service-local suites verify wire behavior and ordered endpoint calls. The central integration test compiles and runs the Go route-plan service, points it at recording HTTP stubs for the unchanged Java services, and then calls it through the real Python travel-plan implementation.

`baseline/mysql57-ddl.sql` is the exact `SHOW CREATE TABLE` oracle captured by running the untouched Java station and consign services against MySQL 5.7. The database-gated replacement tests assert the nullable columns, key name, storage engine, and relevant column order against that oracle.

`baseline/artifacts.json` records SHA-256 and byte size for all four Java reference jars built from the pinned commit. The original Dockerfile base tag (`java:8-jre`) has been removed from Docker Hub, so the record explicitly avoids presenting a substitute JRE image as the historical image digest.

The dependency-lock gate verifies that both Python images use fully pinned, hash-checked requirement files and that both Go images download checksum-verified modules before building in read-only module mode. Generated dependency trees and binary download caches are deliberately excluded from the Voyantclair analysis corpus.

Run all compatibility gates with `make compatibility-test`. The tests do not register duplicate instances in Nacos and therefore cannot accidentally split benchmark traffic during differential testing.
