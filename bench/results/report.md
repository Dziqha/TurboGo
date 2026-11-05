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

| Kasus | Sebelum | Sesudah path fast-path | Delta vs bare |
|-------|---------|------------------------|---------------|
| bare fasthttp (path + switch) | 166 | 166 | — |
| **turbogo param** | 255 | **155** | **-11 (lebih cepat)** |
| fiber v2 | 267 | 267 | +101 |
| fiber v3 | 308 | 308 | +142 |
| turbogo static | 203 | **113** | +71 vs bare tanpa path |
| turbogo fast | 216 | **123** | — |
| turbogo param-json | 721 | **552** | — |

Catatan: baseline statis "bare" (42ns) tidak pernah memanggil `ctx.Path()`,
jadi ia tidak membayar `URI.parse` + `normalizePath`; delta +71 di baris
itu bukan overhead framework, melainkan parsing path yang wajib untuk
router berbasis path. Setelah fast-path, TurboGo bahkan lebih cepat dari
baseline bare yang sama-sama mengakses path (155 vs 166).

### Path parsing fast-path

Profil in-process menunjukkan `URI.parse` + `normalizePath` menyerap
~55% + 32% CPU handler. Untuk mayoritas request target keduanya no-op,
jadi router memakai request target mentah bila normalisasi **terbukti**
tidak akan mengubahnya (tidak ada `%`, `\`, `//`, `/./`, `/..`, byte CTL;
target diawali `/`; query/fragment dipotong):

| | ns/op |
|---|---|
| `ctx.Path()` (parse + normalize) | 92 |
| `routePath` (guard + pakai target mentah) | 14 |
| hemat | **~78 ns/request (-85%)** |

Equivalence dijaga oleh test diferensial 50k target acak
(`app_path_test.go`): bila fast-path dipakai, hasilnya wajib identik
dengan `ctx.Path()`; bila tidak, fallback ke `ctx.Path()`. Tidak ada
mismatch, 23.794 fast / 26.244 fallback.

Efek ke RPS (run ulang, c=200 t=6, median 5): turbogo 447k, fasthttp
449k, fiber3 441k — paritas, urutan tidak berubah. Variasi antar-run
(±15%) jauh lebih besar dari efek ini, konsisten dengan beban yang
syscall-bound.

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


## Addendum v1.1.1 (Go 1.26.5, fasthttp v1.77.0)

Angka di atas diukur pada v1.1.0 (fasthttp 1.62). Setelah bump engine,
pengukuran diulang pada 2026-11-05 di mesin dan `wrk` yang sama.
Metode: round-robin (`fairparams_rr.sh`), median 3 rep, 10s/rep.

### `GET /user/42` — c=512, t=12

| Server | median | min | max | p99 ms | spread |
|---|---|---|---|---|---|
| fasthttp | 454,946 | 451,597 | 456,222 | 4.98 | 1.0% |
| turbogo | 440,657 | 386,537 | 452,579 | 5.14 | 15.0% |
| fiber3 | 428,714 | 406,174 | 444,409 | 5.38 | 8.9% |
| fiber2 | 415,312 | 411,929 | 443,368 | 5.51 | 7.6% |
| stdlib | 287,070 | 269,255 | 288,166 | 10.18 | 6.6% |
| chi | 268,028 | 266,142 | 268,081 | 12.08 | 0.7% |
| echo | 267,032 | 235,743 | 267,744 | 12.16 | 12.0% |
| gin | 252,597 | 239,944 | 263,501 | 14.77 | 9.3% |

### `GET /user/42` — c=200, t=6

| Server | median | min | max | p99 ms | spread |
|---|---|---|---|---|---|
| fasthttp | 456,583 | 450,582 | 481,086 | 2.16 | 6.7% |
| fiber3 | 447,161 | 446,463 | 447,542 | 2.25 | 0.2% |
| fiber2 | 445,616 | 444,756 | 448,179 | 2.20 | 0.8% |
| turbogo | 443,907 | 441,111 | 444,742 | 2.21 | 0.8% |
| stdlib | 288,596 | 287,648 | 301,222 | 3.40 | 4.7% |
| chi | 264,726 | 244,356 | 265,967 | 3.60 | 8.2% |
| gin | 263,915 | 261,501 | 265,697 | 3.63 | 1.6% |
| echo | 254,873 | 252,851 | 255,900 | 3.83 | 1.2% |

### In-process (median 3 x 300k)

| Kasus | ns/req |
|---|---|
| bare fasthttp param | 177 |
| turbogo param | 179 |
| turbogo static | 148 |
| turbogo param+json | 640 |
| fiber2 | 287 |
| fiber3 | 312 |

### Router microbench (`-benchtime=200000x`)

| Benchmark | ns/op | allocs |
|---|---|---|
| Router_Static | 11 | 0 |
| FindInto_ScaleParam | 45 | 0 |
| Router_ScaleParam | 72 | 0 |
| Router_CompiledWildcard | 73 | 0 |
| Router_Wildcard | 76 | 0 |

### Path parsing

`ctx.Path()` 105 ns vs guarded fast path 14 ns (1M iterasi, 0 allocs).

Catatan: angka plaintext/JSON tidak diukur ulang. Keduanya mengukur engine
dan encoder, bukan router; selisih antar server fasthttp berada di dalam
noise, jadi tabel terpisahnya dihapus dari halaman benchmarks.

## Addendum — harness versi baru (Fiber v3 3.6.1, Fiber v2 2.52.16, Echo v5 5.4.4)

Bench module di-update ke rilis terbaru (commit `d71c275`). Library tidak
berubah; semua server pembanding dibangun ulang lalu sweep diulang dengan
metode sama (round-robin, median 3, 10s).

### c=512, t=12

| Server | median | min | max | p99 ms | spread |
|---|---|---|---|---|---|
| fasthttp | 462,022 | 455,666 | 464,543 | 4.97 | 1.9% |
| fiber2 | 453,277 | 445,724 | 456,400 | 5.08 | 2.4% |
| turbogo | 450,054 | 433,408 | 458,268 | 5.07 | 5.5% |
| fiber3 | 441,530 | 440,756 | 450,752 | 5.25 | 2.3% |
| stdlib | 296,132 | 292,545 | 296,678 | 9.46 | 1.4% |
| chi | 275,713 | 271,759 | 277,012 | 11.39 | 1.9% |
| gin | 275,065 | 258,416 | 277,736 | 11.69 | 7.0% |
| echo | 270,836 | 259,125 | 274,009 | 11.26 | 5.5% |

### c=200, t=6

| Server | median | min | max | p99 ms | spread |
|---|---|---|---|---|---|
| fasthttp | 453,821 | 444,840 | 459,026 | 2.20 | 3.1% |
| fiber2 | 446,904 | 440,859 | 450,891 | 2.16 | 2.2% |
| turbogo | 442,483 | 439,822 | 442,856 | 2.20 | 0.7% |
| fiber3 | 439,446 | 431,254 | 443,011 | 2.24 | 2.7% |
| stdlib | 285,608 | 285,363 | 285,989 | 3.52 | 0.2% |
| gin | 264,289 | 262,677 | 265,495 | 3.72 | 1.1% |
| chi | 263,415 | 261,926 | 266,778 | 3.68 | 1.8% |
| echo | 256,752 | 256,026 | 257,498 | 3.66 | 0.6% |

### In-process (median 3 x 300k)

| Kasus | ns/req |
|---|---|
| bare fasthttp param | 173 |
| turbogo param | 180 |
| bare fasthttp static | 63 |
| turbogo static | 144 |
| turbogo fast | 143 |
| turbogo param+json | 605 |
| fiber2 | 287 |
| fiber3 | 305 |

Fiber v3 3.6.1 ~2% lebih cepat dari beta.4 di in-process (305 vs 312 ns)
dan ~3% lebih lambat dari turbogo di wrk; peringkat fasthttp-family tidak
berubah, semuanya di dalam band noise.
