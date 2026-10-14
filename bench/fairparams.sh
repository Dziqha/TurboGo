#!/usr/bin/env bash
set -u
cd "$(dirname "$0")"
declare -A P=( [turbogo]=8080 [fiber2]=8081 [fiber3]=8082 [gin]=8083 [echo]=8084 [chi]=8085 [stdlib]=8086 [fasthttp]=8087 )
order=($(echo "turbogo fiber2 fiber3 gin echo chi stdlib fasthttp" | tr ' ' '\n' | shuf))
OUT="$(pwd)/results/fairparams"
mkdir -p "$OUT"
for s in "${order[@]}"; do
  for r in 1 2 3; do
    PORT=${P[$s]} ./bin/$s >/dev/null 2>&1 &
    pid=$!
    for _ in $(seq 1 40); do
      curl -sf http://127.0.0.1:${P[$s]}/plaintext >/dev/null 2>&1 && break
      sleep 0.2
    done
    sleep 1
    wrk -t12 -c512 -d15s --latency "http://127.0.0.1:${P[$s]}/user/42" > "$OUT/${s}_params_r${r}.txt" 2>&1
    kill $pid 2>/dev/null
    wait $pid 2>/dev/null
    sleep 2
  done
  echo "done $s" >> /tmp/opencode/fairparams.log
done
echo ALL-DONE >> /tmp/opencode/fairparams.log