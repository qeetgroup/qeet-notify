# ── Builder ───────────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG BINARY=server
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /out/qeet-notify-${BINARY} \
    ./cmd/${BINARY}

# ── Runtime ───────────────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

ARG BINARY=server
COPY --from=builder /out/qeet-notify-${BINARY} /qeet-notify
COPY --from=builder /app/migrations /migrations

ENV MIGRATIONS_DIR=/migrations

EXPOSE 8080
ENTRYPOINT ["/qeet-notify"]
