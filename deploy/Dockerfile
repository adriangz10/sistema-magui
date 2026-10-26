# syntax=docker/dockerfile:1
# Imagen del servidor central: API Go + app web compilada, en una imagen mínima sin shell.
# Se construye desde la raíz del repositorio:
#   docker build -f deploy/Dockerfile -t magui-api .

# --- 1. App web (Angular) ---
FROM node:24-alpine AS web
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY frontend/ ./
RUN npm run build

# --- 2. Binarios Go ---
FROM golang:1.27-alpine AS go
ENV CGO_ENABLED=0 GOWORK=off
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
ARG VERSION=dev
RUN go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/api ./cmd/api \
 && go build -trimpath -ldflags "-s -w" -o /out/migrate ./cmd/migrate \
 && go build -trimpath -ldflags "-s -w" -o /out/admin ./cmd/admin

# --- 3. Imagen final ---
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=go /out/ /app/
COPY --from=web /src/frontend/dist/frontend/browser /app/web
ENV APP_ENV=prod \
    HTTP_ADDR=:8080 \
    FRONTEND_DIR=/app/web
EXPOSE 8080
USER nonroot:nonroot
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 CMD ["/app/api", "salud"]
ENTRYPOINT ["/app/api"]
