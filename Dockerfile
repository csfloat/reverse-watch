FROM golang:1.24.0 AS builder

WORKDIR /app
COPY . .
RUN go build -o reverse-watch main.go

# Build the Astro dashboard to web/dist. `npm ci` installs from the
# committed package-lock.json for reproducible, supply-chain-pinned deps.
FROM node:22 AS web-builder

WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM gcr.io/distroless/base-debian12

WORKDIR /app
COPY --from=builder /app/reverse-watch /app/reverse-watch
COPY --from=web-builder /app/web/dist /app/web/dist

EXPOSE 80
CMD ["./reverse-watch"]
