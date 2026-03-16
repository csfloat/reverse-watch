FROM golang:1.24.0 AS builder

WORKDIR /app
COPY . .
RUN go build -o reverse-watch main.go

FROM gcr.io/distroless/base-debian12

WORKDIR /app
COPY --from=builder /app/reverse-watch /app/reverse-watch
COPY --from=builder /app/static/index.html /app/static/index.html

EXPOSE 80
CMD ["./reverse-watch"]
