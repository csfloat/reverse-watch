FROM golang:1.24.0 AS builder

WORKDIR /app
COPY . .
RUN go build -o reverse-watch main.go

FROM gcr.io/distroless/base-debian12

COPY --from=builder /app/reverse-watch /app/reverse-watch
EXPOSE 80
CMD ["./app/reverse-watch"]
