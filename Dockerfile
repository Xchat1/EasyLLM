# Stage 1: Build Vue frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/web

COPY web/package*.json ./
RUN npm install --legacy-peer-deps

COPY web/ ./
RUN npm run build

# Stage 2: Build Go backend
FROM golang:1.25-alpine AS backend-builder
WORKDIR /app

# Install build dependencies for sqlite
RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Copy built frontend
COPY --from=frontend-builder /app/web/dist ./web/dist

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o easyllm .

# Stage 3: Final runtime image
FROM alpine:3.21
WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy binary and static files
COPY --from=backend-builder /app/easyllm .
COPY --from=backend-builder /app/web/dist ./web/dist

# Create data directory
RUN mkdir -p /app/data

# Set timezone
ENV TZ=Asia/Shanghai

# Environment variables (can be overridden)
ENV SERVER_PORT=8080
ENV SERVER_HOST=0.0.0.0
ENV DB_TYPE=sqlite
ENV DB_SQLITE_PATH=/app/data/easyllm.db
ENV DATA_DIR=/app/data

EXPOSE 8080

VOLUME ["/app/data"]

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:8080/api/health || exit 1

CMD ["./easyllm"]
