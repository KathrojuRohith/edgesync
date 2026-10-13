# Fault-injection and durability test plan

Use a disposable local cluster only. Preserve test logs and the exact Git commit used.

## Scenarios

1. **Single-node restart:** write at least 1,000 unique keys, stop the process, restart with the same data directory, and compare all acknowledged values.
2. **Leader termination:** write a sequence of uniquely identified records, terminate the elected leader, wait for a new leader, continue writes, then compare data on all healthy nodes.
3. **Follower isolation:** block a node's Raft links, verify the majority can continue and the minority cannot acknowledge writes, restore links, and verify convergence.
4. **Network delay:** add 100–300 ms RPC delay and record election churn, write latency, timeout rate and recovery time.
5. **Crash during compaction:** repeatedly write/delete keys while killing the process around compaction, restart, and validate tombstone semantics.
6. **Concurrent load:** run concurrent PUT/GET/DELETE/SCAN workloads under `go test -race ./...` and retain output.

## Acceptance criteria

- Every acknowledged write survives a restart when WAL fsync is enabled.
- No two leaders in the same term can commit conflicting entries.
- Minority partitions do not acknowledge writes without a majority.
- Deleted keys remain absent after flush, compaction and restart.
- Race detector passes for the tested workload.

Do not report a scenario as passed until it has been executed and its logs retained. This prototype does not implement Raft snapshots or dynamic membership.
