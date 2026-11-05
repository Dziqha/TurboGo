#!/usr/bin/env python3
import glob
import json
import os
import re

BASE = os.path.dirname(os.path.abspath(__file__))
RAW = os.path.join(BASE, "results", "raw")

FLAGS = re.MULTILINE
LATENCY_RE = re.compile(r"^\s+(\d+)%\s+([0-9.]+)([a-zµ]*)s?$", FLAGS)
RPS_RE = re.compile(r"Requests/sec:\s+([0-9.]+)")
TRANSFER_RE = re.compile(r"Transfer/sec:\s+([0-9.]+)([a-zA-Z]+)")
SOCKET_ERR_RE = re.compile(r"Socket errors:\s+connect\s+(\d+),\s+read\s+(\d+),\s+write\s+(\d+),\s+timeout\s+(\d+)")
NON2XX_RE = re.compile(r"Non-2xx or 3xx responses:\s+(\d+)")
REQS_RE = re.compile(r"^(\d+) requests in ", FLAGS)

def parse_wrk(path):
    with open(path) as f:
        text = f.read()
    result = {
        "rps": None,
        "latency": {},
        "socket_errors": [0, 0, 0, 0],
        "non2xx": 0,
        "total_requests": None,
    }
    m = RPS_RE.search(text)
    if m:
        result["rps"] = float(m.group(1))
    for match in LATENCY_RE.finditer(text):
        pct = int(match.group(1))
        val = float(match.group(2))
        unit = match.group(3)
        if unit.endswith("ms"):
            val /= 1000
        elif unit.endswith("us") or unit.endswith("µs"):
            val /= 1_000_000
        result["latency"][pct] = val
    m = SOCKET_ERR_RE.search(text)
    if m:
        result["socket_errors"] = [int(x) for x in m.groups()]
    m = NON2XX_RE.search(text)
    if m:
        result["non2xx"] = int(m.group(1))
    m = REQS_RE.search(text)
    if m:
        result["total_requests"] = int(m.group(1))
    return result

def main():
    servers = sorted(os.listdir(RAW))
    out = {}
    for name in servers:
        rdir = os.path.join(RAW, name)
        if not os.path.isdir(rdir):
            continue
        out[name] = {}
        for f in sorted(glob.glob(os.path.join(rdir, "*.txt"))):
            base = os.path.basename(f)[:-4]
            if "_c" not in base:
                continue
            ep, c, rep = base.rsplit("_c", 1)[0], base.split("_c")[1].split("_r")[0], base.split("_r")[1]
            entry = parse_wrk(f)
            entry["rep"] = int(rep)
            entry["concurrency"] = int(c)
            out[name].setdefault(ep, []).append(entry)
        rss_path = os.path.join(rdir, "rss.txt")
        if os.path.exists(rss_path):
            with open(rss_path) as f:
                m = re.search(r"VmRSS:\s+(\d+)", f.read())
                out[name]["rss_kb"] = int(m.group(1)) if m else None
        else:
            out[name]["rss_kb"] = None

    def median(vals):
        vals = sorted(vals)
        n = len(vals)
        if n == 0:
            return None
        return vals[n // 2] if n % 2 else (vals[n // 2 - 1] + vals[n // 2]) / 2

    def summarize(entries):
        if not entries:
            return None
        return {
            "rps": median([e["rps"] for e in entries if e["rps"] is not None]),
            "p50_ms": median([e["latency"].get(50) for e in entries if 50 in e["latency"]]),
            "p95_ms": median([e["latency"].get(95) for e in entries if 95 in e["latency"]]),
            "p99_ms": median([e["latency"].get(99) for e in entries if 99 in e["latency"]]),
            "timeouts": sum(e["socket_errors"][3] for e in entries),
            "non2xx": sum(e["non2xx"] for e in entries),
        }

    summary = {}
    for name, eps in out.items():
        summary[name] = {}
        for ep, entries in eps.items():
            if ep == "rss_kb":
                continue
            c512 = [e for e in entries if e["concurrency"] == 512]
            if c512:
                summary[name][ep] = summarize(c512)
        summary[name]["rss_kb"] = out[name].get("rss_kb")

    with open(os.path.join(BASE, "results", "results.json"), "w") as f:
        json.dump({"servers": out, "summary": summary}, f, indent=2)

    print(json.dumps(summary, indent=2))

if __name__ == "__main__":
    main()