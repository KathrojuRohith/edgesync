# EdgeSync

A fault-tolerant distributed key-value store for real-time edge and logistics systems, built from scratch in Go with an LSM-tree storage engine, Raft consensus, a gRPC API and Kubernetes deployment.

M.Tech trimester project.

## Status

| Stage | Component | State |
|---|---|---|
| 1 | Toolchain, repo skeleton | done |
| 2 | Linux profiling baseline | script ready, run it |
| 3 | `kv.Store` contract + in-memory engine + REPL | done |
| 4 | Write-ahead log | next |
| 5–7 | MemTable, SSTable, compaction | pending |
| 8 | Concurrency hardening | pending |
| 9–11 | gRPC, routing, at-least-once | pending |
| 12–14 | Raft | pending |
| 15–19 | Docker, Kubernetes, Helm, Prometheus, Grafana | pending |
| 20–25 | Fault injection, benchmarks, report | pending |

## Requirements

- Linux (Ubuntu 22.04+ or WSL2)
- Go 1.22+
- `make`, `git`
- `sysstat` and `linux-tools-generic` for profiling

## Quickstart

```bash
make test     # unit tests
make race     # race detector — must stay green
make run      # interactive REPL
```

```
edgesync> PUT order:ORD1001 IN_TRANSIT
OK
edgesync> PUT driver:DRV501 AVAILABLE
OK
edgesync> GET order:ORD1001
IN_TRANSIT
edgesync> SCAN order:
order:ORD1001 = IN_TRANSIT
edgesync> DELETE order:ORD1001
OK
edgesync> GET order:ORD1001
(nil)
edgesync> EXIT
```

While the REPL runs, pprof is served at http://127.0.0.1:6060/debug/pprof/

## Layout

```
cmd/edgesync/     node binary
pkg/kv/           storage contract + reference implementation
docs/adr/         architecture decision records
scripts/          profiling and operations scripts
```

Directories for `pkg/lsm`, `internal/raft`, `proto`, `deploy`, `helm` and
`monitoring` are added as each stage begins, not before. An empty directory
in a repository is a promise you have not kept yet.

## Design notes

Start with [ADR 0002](docs/adr/0002-store-interface-as-the-spine.md); it
explains why a five-method interface is the spine of the whole system.
