FROM golang:1.25-bookworm AS builder

# hadolint ignore=DL3008
RUN apt-get update && apt-get install -y --no-install-recommends gcc libc6-dev \
	&& rm -rf /var/lib/apt/lists/*

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
ENV CGO_ENABLED=1
RUN make build

FROM debian:bookworm-slim

# hadolint ignore=DL3008
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates git openssh-client \
	&& rm -rf /var/lib/apt/lists/*

COPY --from=builder /src/bin/octa-agentd /usr/local/bin/octa-agentd
COPY --from=builder /src/bin/octa-agent /usr/local/bin/octa-agent

RUN useradd -m -u 1000 octa
USER octa
WORKDIR /home/octa

ENV HOME=/home/octa
VOLUME ["/home/octa/.config/octaai"]

# Daemon health (optional): map host port to container 8766
EXPOSE 8766

ENTRYPOINT ["octa-agentd"]
