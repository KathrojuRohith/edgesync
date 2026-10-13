# EdgeSync

EdgeSync is a Go-based key-value storage research prototype with a write-ahead log, LSM-tree storage engine, Raft replication mode, HTTP operations console, Prometheus metrics, and local container/Kubernetes deployment examples.

## Components

| Component | Included implementation |
|---|---|
| Storage API | Concurrent byte-oriented key-value contract and reference memory store |
| WAL | CRC-protected append-only records, replay and torn-tail recovery |
| MemTable | Concurrent skip-list write buffer |
| SSTable | Immutable sorted files, sparse index and Bloom filter |
| Compaction | Sorted merge, newest-version wins and tombstone handling |
| Raft | Leader election, append entries, quorum commit, durable term/vote/log and HTTP RPC |
| Replicated store | Raft proposals, apply callback and linearizable read barrier |
| Web console | Responsive key-value workbench, prefix scan, health endpoint and Prometheus metrics |
| Dev tools | HTTP load generator, fault injection CLI and restart durability checker |
| Deployment | Dockerfile, Compose cluster, Kubernetes manifests, Helm chart |
| Observability | Prometheus scrape config and provisioned Grafana dashboard |
| Quality | Go tests, race-test target and GitHub Actions workflow |

## Run locally

Requirements: Go 1.22+, Git, and (for containers) Docker Compose.

```bash
go test ./...
go test -race ./...
make build
make run
```

Open http://localhost:8080. The console provides key-value operations, prefix scans, health at `/healthz`, and metrics at `/metrics`.

## Run a three-node local cluster

```bash
make build
make cluster-up
```

Node consoles are available at http://localhost:8081, http://localhost:8082 and http://localhost:8083. Check process status with `make cluster-status`; stop nodes with `make cluster-down`.

## Docker Compose

```bash
docker compose up --build
```

- Node consoles: ports 8081–8083
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000 (demo credentials: admin / admin)

Change demo credentials before exposing any monitoring service outside a trusted development environment.

## Kubernetes / Helm

The single-node development manifest is in `deploy/k8s`. The Helm chart is in `helm/edgesync`. Build and publish the image to a registry accessible by your cluster before deploying. The included chart defaults to one replica; scaling a stateful Raft cluster requires per-node identities, persistent volumes and explicit peer configuration.

## Validation and limitations

Run `go test ./...`, `go test -race ./...`, `go vet ./...`, and the documented fault scenarios in [docs/FAULT_INJECTION.md](docs/FAULT_INJECTION.md). Record real benchmark output in [docs/BENCHMARK_REPORT.md](docs/BENCHMARK_REPORT.md); no performance numbers are claimed until measured.

This is an educational prototype, not a production-certified database. Raft snapshots and dynamic membership are not implemented. Review consistency, crash recovery, security, access control, backups, observability and resource limits before any production use.
