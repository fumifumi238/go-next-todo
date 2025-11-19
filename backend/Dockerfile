FROM golang:1.25.4-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN ls -R /app
RUN go build -v -o server ./cmd/api
RUN ls -al /app  # 確認用

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/server .
RUN ls -al /app  # 確認用

RUN chmod +x server
CMD ["./server"]
