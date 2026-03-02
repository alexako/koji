# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod ./
# COPY go.sum ./  # uncomment when we have dependencies
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o brain ./cmd/brain

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/brain .

EXPOSE 8080

CMD ["./brain", "-addr", ":8080"]
