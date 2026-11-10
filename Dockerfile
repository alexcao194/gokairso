FROM golang:1.26-alpine

WORKDIR /app

# Install dependencies needed for air and build tools
RUN apk add --no-cache git curl build-base

# Install air for hot reloading and goose for migrations
RUN go install github.com/air-verse/air@latest && \
    go install github.com/pressly/goose/v3/cmd/goose@latest

# Copy go mod files and download dependencies
COPY go.mod go.sum* ./
RUN go mod download

# Note: We don't COPY the rest of the source here because we'll 
# mount it via docker-compose for development.

EXPOSE 8080

CMD ["air", "-c", ".air.toml"]
