# Step 1: Build Go binary
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN apk add --no-cache git && go mod download

COPY . .
RUN go build -o ums-backend main.go

# Step 2: Minimal runtime image
FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/ums-backend .
COPY pkg/migration ./pkg/migration



EXPOSE 8082

CMD ["./ums-backend"]
