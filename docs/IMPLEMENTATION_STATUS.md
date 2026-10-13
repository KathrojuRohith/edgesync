# EdgeSync implementation status

This repository now contains the core source modules and runnable development/deployment scaffolding. The project is a research prototype and should not be described as production-ready.

| Workstream | Repository state | Verification / limitation |
|---|---|---|
| WAL | Implemented | CRC-protected append-only log and torn-tail recovery tests |
| MemTable | Implemented | Concurrent skip-list write buffer |
| SSTable | Implemented | Immutable sorted files, sparse index and Bloom filter |
| Compaction | Implemented | Sorted merge, newest version wins, tombstones handled |
| Concurrency hardening | Partial | Locks and defensive byte copies; run the race suite for each change |
| HTTP API and web console | Implemented | PUT/GET/DELETE, prefix scan, health and metrics endpoints |
| Request retry IDs | Prototype | Optional X-Request-ID deduplication; in-memory dedup table is bounded and not durable across process restart |
| Routing | Implemented for HTTP API | Followers forward API requests when they know the elected leader |
| gRPC transport | Not implemented | The current API and Raft RPC use HTTP/JSON; gRPC/protobuf remains outstanding |
| Raft | Implemented prototype | Election, AppendEntries, quorum commit, durable term/vote/log, leader failover tests; snapshots and dynamic membership are not implemented |
| Docker / Compose | Included | Three-node local topology plus Prometheus and Grafana; run Compose validation locally |
| Kubernetes / Helm | Included as development templates | Helm defaults to one replica; production multi-node StatefulSet identity and peer discovery need work |
| Prometheus / Grafana | Included | Scrape config, datasource provisioning and dashboard |
| Fault injection | Included | `cmd/faultctl` and scenario guide; run scenarios and retain evidence |
| Benchmarks | Included | `cmd/bench` and Go benchmarks; no performance numbers claimed until measured |
| Durability checker | Included | `cmd/checker` verifies writes across restart |

## Verification gates

Before calling this production-ready, run and record:

- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `make build`
- Docker Compose config/build and Kubernetes/Helm template validation
- leader loss, partition, network delay, crash recovery and compaction fault scenarios
- repeatable single-node and three-node latency/throughput measurements

No benchmark results are invented in this repository. Review access control, TLS, resource limits, backup/restore, snapshots, log compaction and operational recovery before production use.
