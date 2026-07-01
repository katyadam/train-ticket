# Train Ticket architecture ground truth

This directory is the machine-readable oracle for the semantics-preserving polyglot benchmark. The `.yaml` files intentionally use JSON syntax, which is valid YAML 1.2, so the verifier has no package dependency.

- `services.yaml` records all benchmark service identities and implementation languages.
- `rest-edges.yaml` records the complete business REST graph. Gateway routing, Nacos, databases, brokers, and telemetry are deliberately excluded from this graph.
- `endpoint-edges.yaml` records method/path-level edges involving the four rewritten services.
- `persistence.yaml` records database ownership and separates infrastructure relations from business edges.
- `scenarios/selected-services.yaml` names representative workloads that activate every selected edge.
- `rest-graph.dot` is the deterministic graph generated from `rest-edges.yaml` and checked in CI.

Run `python3 benchmark-ground-truth/verify.py` from any directory. The verifier rejects duplicate or dangling edges, active Java sources in rewritten services, language mismatches, and added or missing outbound targets in the four replacements. Run `python3 benchmark-ground-truth/generate_graph.py` after an intentional graph change; CI uses `--check` to reject a stale generated graph.

The baseline is Train Ticket commit `313886e99befb94be6cd45f085c98e0019f59829`. Dynamic UUIDs and timestamps may be normalized in differential fixtures; endpoint methods, paths, scalar types, nulls, array order, message punctuation, call order, and call multiplicity may not.
