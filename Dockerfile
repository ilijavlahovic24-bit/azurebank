# ---- build stage ----
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Keširaj dependency sloj
COPY go.mod go.sum ./
RUN go mod download

# Kopiraj izvor i build-uj
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api

# ---- run stage ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /out/api /app/api

EXPOSE 8080
USER nobody:nobody
ENTRYPOINT ["/app/api"]