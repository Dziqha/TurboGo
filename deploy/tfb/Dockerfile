FROM golang:1.25-alpine AS builder

RUN apk --no-cache add git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
COPY turbo-bench/ ./turbo-bench/
COPY core/ ./core/
COPY internal/ ./internal/
COPY middleware/ ./middleware/
RUN CGO_ENABLED=0 go build -o /server -ldflags="-s -w" ./turbo-bench/

FROM alpine:3.21

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /server /server

EXPOSE 8080

CMD ["/server"]
