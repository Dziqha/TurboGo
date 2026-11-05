#!/usr/bin/env python3
import json
import os

BASE = os.path.dirname(os.path.abspath(__file__))
RESULTS = os.path.join(BASE, "results", "results.json")

ENDPOINT_LABELS = {
    "plaintext": "Plaintext",
    "json": "JSON",
    "params": "Dynamic route",
    "db": "DB single query",
    "queries": "DB 20 queries",
    "fortunes": "Fortunes (HTML)",
    "update": "DB 20 updates",
    "plaintextfast": "Plaintext (fast path)",
    "jsonfast": "JSON (fast path)",
}

def fmt_rps(v):
    return f"{v:,.0f}" if v is not None else "-"

def fmt_ms(v):
    return f"{v*1000:.2f}" if v is not None else "-"

def fmt_rss(v):
    return f"{v/1024:.1f} MB" if v else "-"

def main():
    with open(RESULTS) as f:
        data = json.load(f)
    summary = data["summary"]

    servers = list(summary.keys())
    ep_names = [e for e in next(iter(summary.values())).keys() if e != "rss_kb"]

    lines = []
    lines.append("# TurboGo Benchmark Report (Linux, wrk 4.2.0)")
    lines.append("")
    lines.append("- Host: Ryzen 5 5600H (12 threads), CachyOS, Go 1.26.5")
    lines.append("- Load: `wrk -t12 -c512 -d15s --latency`, median dari 3 run (sweep c=1..512)")
    lines.append("- Semua server: GOMAXPROCS=12, mode release, logging dimatikan")
    lines.append("")

    for ep in ep_names:
        if not any(s.get(ep) for s in summary.values()):
            continue
        label = ENDPOINT_LABELS.get(ep, ep)
        lines.append(f"## {label} (`{ep}`)  — RPS @ c=512")
        lines.append("")
        lines.append("| Framework | RPS | p50 (ms) | p95 (ms) | p99 (ms) | Timeouts | Non-2xx |")
        lines.append("|-----------|-----|----------|----------|----------|----------|---------|")
        rows = []
        for s in servers:
            e = summary[s].get(ep)
            if not e or e["rps"] is None:
                continue
            rows.append((e["rps"], s, e))
        rows.sort(reverse=True)
        for rps, s, e in rows:
            lines.append(
                f"| {s} | {fmt_rps(rps)} | {fmt_ms(e['p50_ms'])} | {fmt_ms(e['p95_ms'])} | {fmt_ms(e['p99_ms'])} | {e['timeouts']} | {e['non2xx']} |"
            )
        lines.append("")

    lines.append("## Memory (RSS saat benchmark)")
    lines.append("")
    lines.append("| Framework | RSS |")
    lines.append("|-----------|-----|")
    for s in servers:
        lines.append(f"| {s} | {fmt_rss(summary[s].get('rss_kb'))} |")
    lines.append("")

    lines.append("## Ringkasan peringkat (RPS @ c=512)")
    lines.append("")
    lines.append("| Peringkat | Framework | Menang (endpoint) |")
    lines.append("|-----------|-----------|-------------------|")
    wins = {s: 0 for s in servers}
    for ep in ep_names:
        best = None
        best_rps = -1
        for s in servers:
            e = summary[s].get(ep)
            if e and e["rps"] is not None and e["rps"] > best_rps:
                best_rps = e["rps"]
                best = s
        if best:
            wins[best] += 1
    for i, (s, w) in enumerate(sorted(wins.items(), key=lambda x: -x[1]), 1):
        lines.append(f"| {i} | {s} | {w} |")
    lines.append("")

    with open(os.path.join(BASE, "results", "report.md"), "w") as f:
        f.write("\n".join(lines))
    print(f"report written to {os.path.join(BASE, 'results', 'report.md')}")

if __name__ == "__main__":
    main()