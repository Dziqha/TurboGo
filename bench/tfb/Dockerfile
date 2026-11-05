FROM golang:1.26-alpine AS builder

RUN apk --no-cache add git

WORKDIR /app

COPY go.mod go.sum ./
COPY bench/go.mod bench/go.sum ./bench/
RUN cd bench && go mod download

COPY *.go ./
COPY core/ ./core/
COPY internal/ ./internal/
COPY middleware/ ./middleware/
COPY bench/turbogo/ ./bench/turbogo/
RUN cd bench && CGO_ENABLED=0 go build -o /server -ldflags="-s -w" ./turbogo

FROM alpine:3.21

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /server /server

EXPOSE 8080

CMD ["/server"]
