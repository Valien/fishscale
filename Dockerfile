# Stage 1: Build Svelte frontend
FROM node:22-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.25-alpine AS backend
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
COPY --from=frontend /app/frontend/dist ./internal/frontend/dist
RUN CGO_ENABLED=0 go build -o fishscale ./cmd/fishscale

# Stage 3: Minimal runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S fishscale && adduser -S fishscale -G fishscale
COPY --from=backend /app/fishscale /usr/local/bin/fishscale
RUN mkdir -p /data/photos /data/tsnet-state && \
    chown -R fishscale:fishscale /data
USER fishscale
VOLUME /data
ENTRYPOINT ["fishscale"]
