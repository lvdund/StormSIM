FROM golang:1.25-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN mkdir -p /out \
    && go build -ldflags="-w -s" -o /out/emulator ./cmd/emulator/emulator.go \
    && go build -ldflags="-w -s" -o /out/client ./cmd/client/client.go

FROM debian:bookworm-slim

WORKDIR /app

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        iproute2 \
        iputils-ping \
        lksctp-tools \
        tcpdump \
    && rm -rf /var/lib/apt/lists/*

COPY --from=build /out/emulator /usr/local/bin/emulator
COPY --from=build /out/client /usr/local/bin/client

CMD ["emulator", "-c", "/app/config/config.yml"]
