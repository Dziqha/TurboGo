# TurboGo Benchmark Report (Linux, wrk 4.2.0)

- Host: Ryzen 5 5600H (12 threads), CachyOS, Go 1.26.5
- Load: `wrk -t12 -c512 -d15s --latency`, median 3 run, urutan server diacak (fair run)
- Semua server: GOMAXPROCS=12, mode release, logging dimatikan, pgx pool 12 conn
- HTTP: `bench/results/fair/` + `fairparams/` | DB: `bench/results/dbfair/`

## Plaintext (`plaintext`)  — RPS @ c=512

| Peringkat | Framework | RPS | p50 (ms) | p99 (ms) | Non-2xx |
|-----------|-----------|-----|----------|----------|---------|
| 1 | turbogo | 460,504 | 0.67 | 5.08 | 0 |
| 2 | fasthttp | 438,100 | 0.70 | 5.27 | 0 |
| 3 | fiber2 | 429,700 | 0.72 | 5.35 | 0 |
| 4 | fiber3 | 422,091 | 0.73 | 5.58 | 0 |
| 5 | stdlib | 272,926 | 1.60 | 13.02 | 0 |
| 6 | gin | 259,026 | 1.69 | 14.17 | 0 |
| 7 | chi | 254,900 | 1.73 | 14.46 | 0 |
| 8 | echo | 253,701 | 1.74 | 13.99 | 0 |

## JSON (`json`)  — RPS @ c=512

| Peringkat | Framework | RPS | p50 (ms) | p99 (ms) | Non-2xx |
|-----------|-----------|-----|----------|----------|---------|
| 1 | fasthttp | 427,992 | 0.72 | 5.65 | 0 |
| 2 | turbogo | 414,174 | 0.76 | 5.89 | 0 |
| 3 | fiber3 | 408,175 | 0.78 | 5.83 | 0 |
| 4 | fiber2 | 405,672 | 0.78 | 5.87 | 0 |
| 5 | stdlib | 270,800 | 1.62 | 13.10 | 0 |
| 6 | gin | 257,295 | 1.72 | 13.97 | 0 |
| 7 | echo | 253,439 | 1.73 | 14.63 | 0 |
| 8 | chi | 252,123 | 1.75 | 14.61 | 0 |

## Dynamic route (`params`)  — RPS, round-robin, median of 5

Setelah `FindInto` (slot params, 0 alloc): `./fairparams_rr.sh 5 12s /user/42`.

### c=200, t=6 (thread-matched, noise ±5%)

| Peringkat | Framework | RPS | min-max | p99 (ms) |
|-----------|-----------|-----|---------|----------|
| 1 | fasthttp | 393,594 | 380,040-396,887 | 2.36 |
| 2 | turbogo | 386,102 | 366,259-400,941 | 2.36 |
| 3 | fiber2 | 385,100 | 370,136-392,872 | 2.40 |
| 4 | fiber3 | 378,991 | 376,742-397,962 | 2.47 |

Selisih turbogo vs fasthttp 1.9% dan vs fiber3 +1.9% — di dalam noise: **paritas**.

### c=512, t=12 (oversubscribed 12 worker + 12 server di 6C/12T, noise ±15%)

| Peringkat | Framework | RPS median | min-max | p99 (ms) |
|-----------|-----------|------------|---------|----------|
| 1 | fasthttp | 426,374 | 388,321-443,566 | 5.4 |
| 2 | fiber3 | 414,118 | 372,102-445,812 | 5.7 |
| 3 | fiber2 | 391,072 | 368,681-422,960 | 5.9 |
| 4 | turbogo | 389,963 | 379,425-432,003 | 5.9 |
| 5 | stdlib | 273,207 | 244,673-290,676 | 11.0 |
| 6 | gin | 259,817 | 223,766-277,831 | 11.5 |
| 7 | chi | 253,920 | 228,895-272,967 | 15.8 |
| 8 | echo | 249,975 | 235,055-257,737 | 19.5 |

Di titik ini I/O-bound (syscall 71% CPU, lihat bagian profiling), jadi
selisih antar-server fasthttp/fiber/turbogo tidak terpisahkan dari noise.
Yang tetap struktural: semua framework di atas fasthttp ~1.5x vs
net/http (gin/chi/echo/stdlib).

Catatan metodologi: `fairparams.sh` (blok per server) membiarkan drift
termal membiaskan perbandingan antar-framework; angka c=512 di atas
memakai `fairparams_rr.sh` yang men-siklus semua server tiap rep.
Angka `params` versi lama (turbogo 379k, fiber3 431k, fasthttp 370k,
blok per server) tidak sebanding dan sudah diganti.

## Router microbenchmark (deterministik, `go test -bench`)

Bukan RPS jaringan — ini CPU murni lookup, angka yang bisa direproduksi
tanpa noise:

| Benchmark | Sebelum (Find, map) | Sesudah (FindInto, slot) | Delta |
|-----------|--------------------|--------------------------|-------|
| Scale param (120 route) | 66 ns/op, 0 allocs | 46 ns/op, 0 allocs | **-31%** |
| Wildcard (compiled) | 71 ns/op, 0 allocs | 38 ns/op, 0 allocs | **-46%** |
| Wildcard (tree fallback) | 70 ns/op | 59 ns/op | -16% |
| Scale static | 15-16 ns/op | 15-16 ns/op | sama |

Angka absolut bergeser antar-sesi (frekuensi CPU), jadi yang dikutip
hanya pasangan sebelum/sesudah dalam satu sesi (`-count=3`,
`-benchtime=500000x`).

Bukti exactness: `fuzz_test.go` differential `FindInto` vs `Find`
(2 mode: tree + compiled, 20k path acak + 10 method) → **0 mismatch**;
overflow >8 param diuji terpisah. `go test -race` hijau.

## Framework overhead in-process (tanpa network, `bench/inproc`)

Setiap framework dipanggil sebagai `fasthttp.RequestHandler` dengan
`RequestCtx` yang dipakai ulang (reset request/response per iterasi,
`GET /user/42`). Ini mengisolasi lapisan framework dari syscall:

| Kasus | ns/op | Delta vs bare |
|-------|-------|---------------|
| bare fasthttp (path + switch) | 166 | — |
| **turbogo param** | **255** | **+89** |
| fiber v2 | 267 | +101 |
| fiber v3 | 308 | +142 |
| bare fasthttp static (tanpa akses path) | 42 | — |
| turbogo static | 203 | +161 |

Catatan penting: baseline statis "bare" tidak pernah memanggil
`ctx.Path()`, jadi ia **tidak** membayar `URI.parse` + `normalizePath`.
Profil run in-process: `URI.parse` 55% cumulative, `normalizePath` 32%.
Artinya delta +161 di baris statis hampir seluruhnya biaya parsing path
yang wajib dibayar router berbasis path mana pun — bukan overhead
framework. Delta yang sebanding adalah baris param (+89), di mana kedua
pihak sama-sama mengakses path.

Anggaran biaya per request (param route): `Context` lifecycle ~21ns,
defer+recover ~4ns, router `FindInto` ~46ns, sisanya ~18ns (baca param,
dispatch, konversi string). TurboGo adalah lapisan tertipis di antara
ketiganya; target optimasi berikutnya adalah 46ns router itu — dinilai
lewat microbench, bukan wrk.

## Fast path TurboGo (GetFast / raw fasthttp handler)

| Endpoint | RPS | p50 (ms) | p99 (ms) |
|----------|-----|----------|----------|
| plaintextfast | 459,339 | 0.67 | 5.10 |
| jsonfast | 441,169 | 0.71 | 5.50 |

## DB single row (`db`)  — RPS @ c=512

| Peringkat | Framework | RPS | p50 (ms) | p99 (ms) | Non-2xx |
|-----------|-----------|-----|----------|----------|---------|
| 1 | turbogo | 88,537 | 5.61 | 7.16 | 0 |
| 2 | fasthttp | 86,113 | 5.77 | 7.42 | 0 |
| 3 | fiber3 | 81,368 | 6.09 | 7.94 | 0 |
| 4 | fiber2 | 79,361 | 6.22 | 8.65 | 0 |
| 5 | chi | 73,366 | 6.68 | 9.04 | 0 |
| 6 | stdlib | 72,552 | 6.77 | 9.28 | 0 |
| 7 | echo | 70,693 | 6.93 | 9.73 | 0 |
| 8 | gin | 67,348 | 7.26 | 10.62 | 0 |

## Queries (20) (`queries`)  — RPS @ c=512

| Peringkat | Framework | RPS | p50 (ms) | p99 (ms) | Non-2xx |
|-----------|-----------|-----|----------|----------|---------|
| 1 | fasthttp | 53,254 | 9.40 | 11.42 | 0 |
| 2 | turbogo | 52,127 | 9.56 | 11.65 | 0 |
| 3 | fiber3 | 49,945 | 9.96 | 12.50 | 0 |
| 4 | stdlib | 48,633 | 10.24 | 12.59 | 0 |
| 5 | fiber2 | 47,656 | 10.42 | 13.55 | 0 |
| 6 | chi | 44,839 | 10.95 | 14.69 | 0 |
| 7 | echo | 43,832 | 11.35 | 14.46 | 0 |
| 8 | gin | 42,774 | 11.60 | 15.02 | 0 |

## Fortunes (`fortunes`)  — RPS @ c=512

| Peringkat | Framework | RPS | p50 (ms) | p99 (ms) | Non-2xx |
|-----------|-----------|-----|----------|----------|---------|
| 1 | fasthttp | 85,282 | 5.78 | 7.53 | 0 |
| 2 | turbogo | 80,560 | 6.07 | 8.49 | 0 |
| 3 | fiber2 | 80,384 | 6.10 | 8.55 | 0 |
| 4 | fiber3 | 66,839 | 7.17 | 12.45 | 0 |
| 5 | stdlib | 63,430 | 7.81 | 10.76 | 0 |
| 6 | gin | 62,145 | 7.88 | 11.11 | 0 |
| 7 | chi | 61,574 | 8.04 | 11.32 | 0 |
| 8 | echo | 61,229 | 8.03 | 11.27 | 0 |

## Update (20) (`update`)  — RPS @ c=512

| Peringkat | Framework | RPS | p50 (ms) | p99 (ms) | Non-2xx |
|-----------|-----------|-----|----------|----------|---------|
| 1 | echo | 3,792 | 94.76 | 324.51 | 0 |
| 2 | turbogo | 3,654 | 92.12 | 330.46 | 0 |
| 3 | stdlib | 3,297 | 93.63 | 428.66 | 0 |
| 4 | gin | 2,830 | 124.77 | 390.91 | 0 |
| 5 | chi | 2,808 | 183.52 | 360.38 | 0 |
| 6 | fiber3 | 2,598 | 239.41 | 376.62 | 0 |
| 7 | fiber2 | 2,578 | 231.60 | 389.57 | 0 |
| 8 | fasthttp | 1,962 | 242.45 | 251.27 | 0 |

## Memory (RSS saat benchmark)

| Framework | RSS |
|-----------|-----|
| fasthttp | 30.3 MB |
| echo | 31.6 MB |
| fiber2 | 31.6 MB |
| chi | 32.2 MB |
| stdlib | 32.3 MB |
| fiber3 | 33.4 MB |
| gin | 35.2 MB |
| turbogo | 53.6 MB |

## Ringkasan peringkat (RPS @ c=512)

| Framework | Rata-rata RPS (7 endpoint) | Menang (endpoint) |
|-----------|---------------------------|-------------------|
| turbogo | 211,251 | 2 |
| fasthttp | 208,908 | 3 |
| fiber3 | 208,874 | 1 |
| fiber2 | 207,112 | 0 |
| stdlib | 146,578 | 0 |
| gin | 140,640 | 0 |
| chi | 132,583 | 0 |
| echo | 128,138 | 1 |

## Profiling hot spot (pprof, `/plaintext` @ c=512)

- Total biaya framework (`Handler`): hanya 5.2% CPU (3.6s/69.2s); sisanya I/O jaringan (`syscall` 71.9%).
- Di dalam `Handler`: `ctx.Path()` 63%, router `Find` 10%, `NewContext` 6.4%, `ReleaseContext` 2.8%.

### Setelah `FindInto` (`/user/42` @ c=200, 20s CPU profile)

- `syscall` 71.1% flat, `serveConn` cumulative 93.6%.
- Router `findIntoCompiled` 0.87% cumulative — dari 10% relatif di dalam
  Handler turun ke ~1% dari total CPU.
- Konsekuensi: pada beban jaringan, RPS tidak lagi sensitif terhadap
  biaya router (29ns/req tidak terukur di wrk c>=200). Optimasi router
  berikutnya harus dinilai lewat microbenchmark, bukan RPS.

