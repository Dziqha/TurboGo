#!/usr/bin/env bash
# Round-robin route comparison. Every rep cycles through all servers, so
# thermal/background drift hits each framework evenly (fairparams.sh runs
# blocks per server, which lets drift bias cross-server comparisons).
#
# Usage: ./fairparams_rr.sh [reps] [duration] [endpoint]
# Env:   SERVERS="turbogo fiber3 fasthttp"  subset
#        WRK_T=4 WRK_C=100                  load shape (default 12/512)
#        OUT=results/rrparams               output dir
set -u
cd "$(dirname "$0")"

REPS=${1:-5}
DUR=${2:-12s}
EP=${3:-/user/42}
WRK_T=${WRK_T:-12}
WRK_C=${WRK_C:-512}
OUT=${OUT:-results/rrparams}

declare -A P=( [turbogo]=8080 [fiber2]=8081 [fiber3]=8082 [gin]=8083 [echo]=8084 [chi]=8085 [stdlib]=8086 [fasthttp]=8087 )
ALL=(turbogo fiber2 fiber3 gin echo chi stdlib fasthttp)
if [ -n "${SERVERS:-}" ]; then
  read -r -a SERVERS <<< "$SERVERS"
else
  SERVERS=("${ALL[@]}")
fi

mkdir -p "$OUT"
rm -f "$OUT"/*_r*.txt

for r in $(seq 1 "$REPS"); do
  for s in "${SERVERS[@]}"; do
    PORT=${P[$s]} ./bin/$s >/dev/null 2>&1 &
    pid=$!
    for _ in $(seq 1 40); do
      curl -sf "http://127.0.0.1:${P[$s]}/plaintext" >/dev/null 2>&1 && break
      sleep 0.2
    done
    sleep 1
    wrk -t"$WRK_T" -c"$WRK_C" -d"$DUR" --latency "http://127.0.0.1:${P[$s]}$EP" \
      > "$OUT/${s}_r${r}.txt" 2>&1
    kill $pid 2>/dev/null
    wait $pid 2>/dev/null
    sleep 1
  done
  echo "rep $r done"
done

python3 - "$OUT" <<'PY'
import glob, os, re, statistics, sys
out = sys.argv[1]
rows = []
for s in sorted({os.path.basename(p).rsplit("_r", 1)[0] for p in glob.glob(out + "/*.txt")}):
    vals, p99s = [], []
    for p in glob.glob(f"{out}/{s}_r*.txt"):
        txt = open(p).read()
        m = re.search(r"Requests/sec:\s+([\d.]+)", txt)
        if m:
            vals.append(float(m.group(1)))
        p99 = re.search(r"99%\s+([\d.]+)(us|ms|s)", txt)
        if p99:
            v, u = float(p99.group(1)), p99.group(2)
            p99s.append(v * {"us": 1e-3, "ms": 1.0, "s": 1e3}[u])
    if vals:
        rows.append((statistics.median(vals), s, min(vals), max(vals),
                     statistics.median(p99s) if p99s else 0, len(vals)))
rows.sort(reverse=True)
print(f"{'rank':<5}{'server':<12}{'median':>10}{'min':>10}{'max':>10}{'p99ms':>9}{'spread':>9}{'n':>4}")
for i, (med, s, lo, hi, p99, n) in enumerate(rows, 1):
    spread = (hi - lo) / med * 100
    print(f"{i:<5}{s:<12}{med:>10.0f}{lo:>10.0f}{hi:>10.0f}{p99:>9.2f}{spread:>8.1f}%{n:>4}")
PY
