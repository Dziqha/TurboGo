#!/usr/bin/env python3
import glob, re, statistics

d = {}
for f in glob.glob('results/fair/*.txt'):
    name = f.split('/')[-1][:-4]
    server_ep, r = name.rsplit('_r', 1)
    s, ep = server_ep.rsplit('_', 1)
    t = open(f).read()
    rps = float(re.search(r'Requests/sec:\s+([0-9.]+)', t).group(1))
    lat = {}
    for m in re.finditer(r'^\s+(\d+)%\s+([0-9.]+)([a-zµ]+)$', t, re.M):
        pct = int(m.group(1)); v = float(m.group(2)); u = m.group(3)
        if u.endswith('ms'): v /= 1000
        elif u.endswith('us'): v /= 1e6
        lat[pct] = v
    d.setdefault((s, ep), []).append((rps, lat))

rows = {}
for (s, ep), vals in d.items():
    rows.setdefault(ep, []).append((s, statistics.median(x[0] for x in vals),
                                    statistics.median(x[1].get(50, 0) for x in vals) * 1000,
                                    statistics.median(x[1].get(99, 0) for x in vals) * 1000))

fast = {'plaintextfast': (459339.16, 0.67, 5.10), 'jsonfast': (441168.83, 0.71, 5.50)}

def fmt(v):
    return f"{v:,.0f}"

out = []
out.append("# TurboGo Benchmark Report (Linux, wrk 4.2.0)")
out.append("")
out.append("- Host: Ryzen 5 5600H (12 threads), CachyOS, Go 1.26.5")
out.append("- Load: `wrk -t12 -c512 -d15s --latency`, median 3 run, urutan server diacak (fair run)")
out.append("- Semua server: GOMAXPROCS=12, mode release, logging dimatikan")
out.append("")

for ep, title in [('plaintext', 'Plaintext'), ('json', 'JSON'), ('params', 'Dynamic route')]:
    out.append(f"## {title} (`{ep}`)  — RPS @ c=512")
    out.append("")
    out.append("| Peringkat | Framework | RPS | p50 (ms) | p99 (ms) |")
    out.append("|-----------|-----------|-----|----------|----------|")
    for i, (s, rps, p50, p99) in enumerate(sorted(rows[ep], key=lambda x: -x[1]), 1):
        out.append(f"| {i} | {s} | {fmt(rps)} | {p50:.2f} | {p99:.2f} |")
    out.append("")

out.append("## Fast path TurboGo (GetFast / raw fasthttp handler)")
out.append("")
out.append("| Endpoint | RPS | p50 (ms) | p99 (ms) |")
out.append("|----------|-----|----------|----------|")
for ep, (rps, p50, p99) in fast.items():
    out.append(f"| {ep} | {fmt(rps)} | {p50:.2f} | {p99:.2f} |")
out.append("")

out.append("## Memory (RSS saat benchmark sweep awal)")
out.append("")
out.append("| Framework | RSS |")
out.append("|-----------|-----|")
for s, rss in [('fasthttp', '30.3 MB'), ('echo', '31.6 MB'), ('fiber2', '31.6 MB'), ('chi', '32.2 MB'),
               ('stdlib', '32.3 MB'), ('fiber3', '33.4 MB'), ('gin', '35.2 MB'), ('turbogo', '53.6 MB')]:
    out.append(f"| {s} | {rss} |")
out.append("")

wins = {}
for ep, data in rows.items():
    wins[max(data, key=lambda x: x[1])[0]] = wins.get(max(data, key=lambda x: x[1])[0], 0) + 1
out.append("## Ringkasan peringkat (RPS @ c=512)")
out.append("")
out.append("| Framework | Menang (endpoint) | Rata-rata posisi |")
out.append("|-----------|-------------------|------------------|")
rank = {}
for ep, data in rows.items():
    for i, (s, _, _, _) in enumerate(sorted(data, key=lambda x: -x[1]), 1):
        rank.setdefault(s, []).append(i)
for s, positions in sorted(rank.items(), key=lambda x: statistics.mean(x[1])):
    out.append(f"| {s} | {wins.get(s, 0)} | {statistics.mean(positions):.2f} |")
out.append("")

out.append("## Profiling hot spot (pprof, `/plaintext` @ c=512)")
out.append("")
out.append("- Total biaya framework (`Handler`): hanya 5.2% CPU (3.6s/69.2s); sisanya I/O jaringan (`syscall` 71.9%).")
out.append("- Di dalam `Handler`: `ctx.Path()` 63%, router `Find` 10%, `NewContext` 6.4%, `ReleaseContext` 2.8%.")
out.append("")

with open('results/report.md', 'w') as f:
    f.write('\n'.join(out) + '\n')
print('ok')