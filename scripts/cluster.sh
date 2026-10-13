#!/usr/bin/env bash
# Start and control a local three-node Raft cluster. Use only for development.
set -euo pipefail
N=${NODES:-3}; BASE=${BASE_PORT:-8081}; DATA=${DATA_DIR:-./data}; BIN=./bin/edgesync
peers(){ local s=""; for i in $(seq 1 "$N"); do s+="node-$i=localhost:$((BASE+i-1)),"; done; echo "${s%,}"; }
start_one(){ local i=$1; mkdir -p "$DATA/node-$i"; nohup "$BIN" -id "node-$i" -http ":$((BASE+i-1))" -data "$DATA/node-$i" -peers "$(peers)" -pprof "" > "$DATA/node-$i.log" 2>&1 & echo $! > "$DATA/node-$i.pid"; echo "started node-$i on :$((BASE+i-1))"; }
case "${1:-}" in
 start) [[ -x $BIN ]] || { echo "build first: make build"; exit 1; }; mkdir -p "$DATA"; for i in $(seq 1 "$N"); do start_one "$i"; done; echo "console: http://localhost:$BASE" ;;
 stop) for i in $(seq 1 "$N"); do f="$DATA/node-$i.pid"; if [[ -f $f ]]; then kill "$(cat "$f")" 2>/dev/null || true; rm -f "$f"; fi; done; echo stopped ;;
 status) for i in $(seq 1 "$N"); do f="$DATA/node-$i.pid"; if [[ -f $f ]] && kill -0 "$(cat "$f")" 2>/dev/null; then echo "node-$i running"; else echo "node-$i stopped"; fi; done ;;
 kill) i=${2:?node number required}; f="$DATA/node-$i.pid"; [[ -f $f ]] && kill -9 "$(cat "$f")" && echo "killed node-$i" ;;
 restart) start_one "${2:?node number required}" ;;
 clean) rm -rf "$DATA" ;;
 *) echo "usage: $0 start|stop|status|clean|kill N|restart N"; exit 2 ;;
esac
