APP_NAME=TurboGo
PKG_PATH=./test
CPU_PROF=cpu.prof
MEM_PROF=mem.prof
BENCH_FUNC=.
COVER_FILE=coverage.out

.PHONY: all test bench profile clean cover show-cover

# Jalankan semua test
test:
	go test $(PKG_PATH) -v

# Jalankan benchmark biasa
bench:
	go test $(PKG_PATH) -bench=$(BENCH_FUNC) -benchmem

# Jalankan benchmark dengan profiling
profile:
	go test $(PKG_PATH) -bench=$(BENCH_FUNC) -cpuprofile=$(CPU_PROF) -memprofile=$(MEM_PROF)

# Tampilkan profil CPU (harus install graphviz untuk web)
cpu-prof:
	go tool pprof $(CPU_PROF)

# Tampilkan profil Memori
mem-prof:
	go tool pprof $(MEM_PROF)

# Jalankan test + coverage
cover:
	go test $(PKG_PATH) -coverprofile=$(COVER_FILE)
	go tool cover -func=$(COVER_FILE)

# Tampilkan coverage dalam bentuk HTML (buka di browser)
show-cover:
	go tool cover -html=$(COVER_FILE)

# Hapus file hasil profil dan coverage
clean:
	-del $(CPU_PROF) 2>nul || rm -f $(CPU_PROF)
	-del $(MEM_PROF) 2>nul || rm -f $(MEM_PROF)
	-del $(COVER_FILE) 2>nul || rm -f $(COVER_FILE)
