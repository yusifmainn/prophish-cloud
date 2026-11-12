# Multi-stage build. Produces a tiny, non-root, shell-less runtime image.
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# -trimpath removes local filesystem paths from the binary; static build (no libc).
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/prophish ./cmd/prophish
# cloudflared ships per-campaign quick tunnels (official static binary; spawned
# via exec.Command, no shell needed in the runtime image). CA certs in the final
# image already cover its HTTPS control connection (same trust store as SMTP).
ADD https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64 /out/cloudflared
RUN chmod +x /out/cloudflared

# distroless/static: no shell, no package manager, runs as non-root, ships CA certs
# (needed for SMTP STARTTLS to external mail servers).
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/prophish /prophish
COPY --from=build /out/cloudflared /usr/local/bin/cloudflared
# Embedded HTML templates are compiled into the binary, so no extra files are needed.
EXPOSE 8443 8080
ENTRYPOINT ["/prophish"]
