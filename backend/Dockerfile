FROM golang:1.25.4-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -v -o server ./cmd/api

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/server .

RUN chmod +x server
CMD ["./server"]
