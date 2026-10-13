# EdgeSync implementation status

This repository is a research prototype. The source tree is organized around a byte-oriented key-value contract, an LSM storage engine, a Raft replication layer, an HTTP console/API, and local observability/deployment tooling.

## Implemented in the source project

- LSM engine components: write-ahead log, concurrent skip-list MemTable, sorted SSTable files, Bloom filters, and merge/compaction utilities.
- Raft: leader election, log replication, durable term/vote/log storage, HTTP transport, and tests.
- Replicated KV adapter: writes proposed through Raft, read barrier, and bounded request-ID deduplication.
- HTTP console and REST endpoints, node status, health, and Prometheus metrics.
- Developer tools: benchmark client, fault-injection controller, durability checker, cluster script, and Go tests.
- Deployment/observability: Dockerfile, Compose topology, Prometheus scrape configuration, Grafana provisioning and dashboard.

## Verification still required before production use

Run the full Go test suite and race detector, build all command binaries, validate Compose and Kubernetes manifests in the target environment, and capture benchmark/fault-injection results. This project is an educational prototype, not a production-certified database. Consensus, durability, security, backup/restore, resource limits, and recovery behavior require independent review.
