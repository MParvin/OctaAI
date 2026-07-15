FROM golang:1.23-bookworm AS builder

RUN apt-get update && apt-get install -y gcc libc6-dev && rm -rf /var/lib/apt/lists/*

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
ENV CGO_ENABLED=1
RUN make build

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates git openssh-client && rm -rf /var/lib/apt/lists/*

COPY --from=builder /src/bin/octa-agentd /usr/local/bin/octa-agentd
COPY --from=builder /src/bin/octa-agent /usr/local/bin/octa-agent

RUN useradd -m -u 1000 octa
USER octa
WORKDIR /home/octa

ENV HOME=/home/octa
VOLUME ["/home/octa/.config/octaai"]

ENTRYPOINT ["octa-agentd"]
