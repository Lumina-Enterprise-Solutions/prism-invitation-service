# Tahap 1: Builder
FROM golang:1.24-alpine AS builder
ENV CGO_ENABLED=0
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags="-w -s" -o /app/server .

# Tahap 2: Final Image
FROM alpine:latest
WORKDIR /app
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
COPY --from=builder /app/server /app/server
LABEL org.opencontainers.image.source="https://github.com/Lumina-Enterprise-Solutions/prism-invitation-service"
LABEL org.opencontainers.image.title="PrismInvitationService"
LABEL org.opencontainers.image.description="Service for managing user invitations."
RUN chown -R appuser:appgroup /app
USER appuser
EXPOSE 8080
CMD ["./server"]
