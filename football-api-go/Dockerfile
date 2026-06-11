# ── development stage ──────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS dev
RUN apk add --no-cache git ca-certificates tzdata
RUN go install github.com/air-verse/air@latest
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
CMD ["air", "-c", ".air.toml"]

# ── builder stage ───────────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder
RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server/main.go

# ── production stage ────────────────────────────────────────────────────────────
FROM scratch AS production
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
