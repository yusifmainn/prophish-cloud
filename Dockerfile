# Multi-stage build. Produces a tiny, non-root, shell-less runtime image.
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# -trimpath removes local filesystem paths from the binary; static build (no libc).
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/secdrill ./cmd/secdrill

# distroless/static: no shell, no package manager, runs as non-root, ships CA certs
# (needed for SMTP STARTTLS to external mail servers).
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/secdrill /secdrill
# Embedded HTML templates are compiled into the binary, so no extra files are needed.
EXPOSE 8443 8080
ENTRYPOINT ["/secdrill"]
