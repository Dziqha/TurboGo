module github.com/Dziqha/TurboGo

go 1.26.0

require (
	github.com/go-playground/validator/v10 v10.30.6
	github.com/valyala/fasthttp v1.77.0
)

require (
	github.com/gabriel-vasile/mimetype v1.4.16 // indirect
	github.com/go-playground/locales v0.14.2 // indirect
	github.com/go-playground/universal-translator v0.18.2 // indirect
	github.com/leodido/go-urn v1.5.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/molecule-man/go-brrr v1.3.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/crypto v0.58.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
)

require (
	github.com/fatih/color v1.18.0
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/klauspost/compress v1.20.3 // indirect
	github.com/stretchr/testify v1.12.1
	github.com/valyala/bytebufferpool v1.0.0 // indirect
)

// v1.0.x predates compiled routing, zero-allocation params and the path
// fast-path. Retracted so tooling steers new work to v1.1.0.
retract [v1.0.0, v1.0.3]
