#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

WRK=${WRK:-wrk}
DUR=${DUR:-15s}
THREADS=${THREADS:-12}
REPS=3

RAW=results/raw
LOGS=results/logs
mkdir -p "$RAW" "$LOGS"

declare -A PORTS=(
  [turbogo]=8080
  [fiber2]=8081
  [fiber3]=8082
  [gin]=8083
  [echo]=8084
  [chi]=8085
  [stdlib]=8086
  [fasthttp]=8087
)

declare -A ENDPOINTS=(
  [plaintext]="/plaintext"
  [json]="/json"
  [params]="/user/42"
  [db]="/db"
  [queries]="/queries?queries=20"
  [fortunes]="/fortunes"
  [update]="/update?queries=20"
)

declare -A TURBO_EXTRA=(
  [plaintextfast]="/plaintextfast"
  [jsonfast]="/jsonfast"
)

DB_UP=0
if pg_isready -q -h localhost -p 5432 2>/dev/null; then
  DB_UP=1
fi

run_wrk() { # name endpoint url conc rep
  local name=$1 ep=$2 url=$3 conc=$4 rep=$5
  local out="$RAW/$name/${ep}_c${conc}_r${rep}.txt"
  "$WRK" -t"$THREADS" -c"$conc" -d"$DUR" --latency --timeout 5s \
    "http://127.0.0.1:${PORTS[$name]}$url" >"$out" 2>&1 || true
}

bench_server() { # name
  local name=$1 port=${PORTS[$1]}
  local pid
  rm -rf "$RAW/$name"
  mkdir -p "$RAW/$name"

  PORT="$port" "./bin/$name" >"$LOGS/$name.log" 2>&1 &
  pid=$!

  for _ in $(seq 1 50); do
    if curl -sf "http://127.0.0.1:$port/plaintext" >/dev/null 2>&1; then
      break
    fi
    if ! kill -0 "$pid" 2>/dev/null; then
      echo "FAIL: $name crashed on startup"
      tail -5 "$LOGS/$name.log"
      return 1
    fi
    sleep 0.2
  done

  for ep in plaintext json params; do
    for conc in 1 8 32 128 512; do
      run_wrk "$name" "$ep" "${ENDPOINTS[$ep]}" "$conc" 1
    done
    for r in $(seq 1 "$REPS"); do
      run_wrk "$name" "$ep" "${ENDPOINTS[$ep]}" 512 "$r"
    done
  done

  if [ "$DB_UP" = 1 ]; then
    for ep in db queries fortunes update; do
      for r in $(seq 1 "$REPS"); do
        run_wrk "$name" "$ep" "${ENDPOINTS[$ep]}" 512 "$r"
      done
    done
  fi

  if [ "$name" = "turbogo" ]; then
    for ep in "${!TURBO_EXTRA[@]}"; do
      for r in $(seq 1 "$REPS"); do
        run_wrk "$name" "$ep" "${TURBO_EXTRA[$ep]}" 512 "$r"
      done
    done
  fi

  grep VmRSS /proc/$pid/status >"$RAW/$name/rss.txt" 2>/dev/null || true
  kill "$pid" 2>/dev/null || true
  wait "$pid" 2>/dev/null || true
  echo "done: $name"
}

order=(turbogo fiber2 fiber3 gin echo chi stdlib fasthttp)
for s in "${order[@]}"; do
  bench_server "$s" || true
done

python3 aggregate.py
echo "== ALL DONE =="