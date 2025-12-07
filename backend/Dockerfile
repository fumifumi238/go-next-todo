FROM golang:1.23.12-alpine3.22 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -v -o server ./cmd/api

FROM alpine:3.22
WORKDIR /app
COPY --from=builder /app/server .

RUN chmod +x server
CMD ["./server"]
