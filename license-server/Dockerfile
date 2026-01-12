FROM golang:alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go.mod and go.sum
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build
RUN go mod tidy
RUN go build -o license-server main.go

# Runtime
FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/license-server .
COPY --from=builder /app/static ./static

EXPOSE 8081
CMD ["./license-server"]
