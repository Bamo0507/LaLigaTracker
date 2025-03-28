FROM golang:1.24-alpine as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main .

FROM alpine:3.17
WORKDIR /app

COPY --from=builder /app/main .

EXPOSE 8081
CMD ["./main"]