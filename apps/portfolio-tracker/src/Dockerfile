# apps/portfolio-tracker/src/Dockerfile
# Multi-stage build — final image ~15MB

FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /portfolio-tracker .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /portfolio-tracker /app/portfolio-tracker
EXPOSE 8080
ENTRYPOINT ["/app/portfolio-tracker"]
