#!/usr/bin/env python3
import glob, re, statistics

def load(dir):
    d = {}
    for f in glob.glob(f'{dir}/*.txt'):
        name = f.split('/')[-1][:-4]
        server_ep, r = name.rsplit('_r', 1)
        s, ep = server_ep.rsplit('_', 1)
        t = open(f).read()
        m = re.search(r'Requests/sec:\s+([0-9.]+)', t)
        if not m:
            continue
        rps = float(m.group(1))
        non2xx = re.search(r'Non-2xx or 3xx responses:\s+(\d+)', t)
        n2 = int(non2xx.group(1)) if non2xx else 0
        lat = {}
        for mm in re.finditer(r'^\s+(\d+)%\s+([0-9.]+)([a-zµ]+)$', t, re.M):
            pct = int(mm.group(1)); v = float(mm.group(2)); u = mm.group(3)
            if u.endswith('ms'): v /= 1000
            elif u.endswith('us'): v /= 1e6
            lat[pct] = v
        d.setdefault((s, ep), []).append((rps, lat, n2))
    return d

def table(title, ep, d):
    rows = []
    for (s, e), vals in d.items():
        if e != ep: continue
        rows.append((s,
                     statistics.median(x[0] for x in vals),
                     statistics.median(x[1].get(50, 0) for x in vals) * 1000,
                     statistics.median(x[1].get(99, 0) for x in vals) * 1000,
                     sum(x[2] for x in vals)))
    rows.sort(key=lambda x: -x[1])
    out = [f"## {title} (`{ep}`)  — RPS @ c=512", ""]
    out.append("| Peringkat | Framework | RPS | p50 (ms) | p99 (ms) | Non-2xx |")
    out.append("|-----------|-----------|-----|----------|----------|---------|")
    for i, (s, rps, p50, p99, n2) in enumerate(rows, 1):
        out.append(f"| {i} | {s} | {rps:,.0f} | {p50:.2f} | {p99:.2f} | {n2} |")
    out.append("")
    return out, rows

httpd = load('results/fair')
httpd.update(load('results/fairparams'))
dbd = load('results/dbfair')

out = []
out.append("# TurboGo Benchmark Report (Linux, wrk 4.2.0)")
out.append("")
out.append("- Host: Ryzen 5 5600H (12 threads), CachyOS, Go 1.26.5")
out.append("- Load: `wrk -t12 -c512 -d15s --latency`, median 3 run, urutan server diacak (fair run)")
out.append("- Semua server: GOMAXPROCS=12, mode release, logging dimatikan, pgx pool 12 conn")
out.append("- HTTP: `bench/results/fair/` + `fairparams/` | DB: `bench/results/dbfair/`")
out.append("")

for ep, title in [('plaintext', 'Plaintext'), ('json', 'JSON'), ('params', 'Dynamic route')]:
    t, _ = table(title, ep, httpd)
    out += t

out.append("## Fast path TurboGo (GetFast / raw fasthttp handler)")
out.append("")
out.append("| Endpoint | RPS | p50 (ms) | p99 (ms) |")
out.append("|----------|-----|----------|----------|")
for ep, (rps, p50, p99) in [('plaintextfast', (459339, 0.67, 5.10)), ('jsonfast', (441169, 0.71, 5.50))]:
    out.append(f"| {ep} | {rps:,} | {p50:.2f} | {p99:.2f} |")
out.append("")

for ep, title in [('db', 'DB single row'), ('queries', 'Queries (20)'), ('fortunes', 'Fortunes'), ('update', 'Update (20)')]:
    t, _ = table(title, ep, dbd)
    out += t

out.append("## Memory (RSS saat benchmark)")
out.append("")
out.append("| Framework | RSS |")
out.append("|-----------|-----|")
for s, rss in [('fasthttp', '30.3 MB'), ('echo', '31.6 MB'), ('fiber2', '31.6 MB'), ('chi', '32.2 MB'),
               ('stdlib', '32.3 MB'), ('fiber3', '33.4 MB'), ('gin', '35.2 MB'), ('turbogo', '53.6 MB')]:
    out.append(f"| {s} | {rss} |")
out.append("")

all_rows = {}
for ep, d in [('plaintext', httpd), ('json', httpd), ('params', httpd),
              ('db', dbd), ('queries', dbd), ('fortunes', dbd), ('update', dbd)]:
    _, rows = table('', ep, d)
    for s, rps, p50, p99, n2 in rows:
        all_rows.setdefault(s, {})[ep] = rps

endpoints = ['plaintext', 'json', 'params', 'db', 'queries', 'fortunes', 'update']
wins = {s: 0 for s in all_rows}
for ep in endpoints:
    best = max(all_rows, key=lambda s: all_rows[s].get(ep, -1))
    wins[best] += 1

out.append("## Ringkasan peringkat (RPS @ c=512)")
out.append("")
out.append("| Framework | Rata-rata RPS (7 endpoint) | Menang (endpoint) |")
out.append("|-----------|---------------------------|-------------------|")
for s in sorted(all_rows, key=lambda x: -statistics.mean(all_rows[x].values())):
    out.append(f"| {s} | {statistics.mean(all_rows[s].values()):,.0f} | {wins[s]} |")
out.append("")

out.append("## Profiling hot spot (pprof, `/plaintext` @ c=512)")
out.append("")
out.append("- Total biaya framework (`Handler`): hanya 5.2% CPU (3.6s/69.2s); sisanya I/O jaringan (`syscall` 71.9%).")
out.append("- Di dalam `Handler`: `ctx.Path()` 63%, router `Find` 10%, `NewContext` 6.4%, `ReleaseContext` 2.8%.")
out.append("")

with open('results/report.md', 'w') as f:
    f.write('\n'.join(out) + '\n')
print('report written')