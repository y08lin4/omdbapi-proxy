FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod ./
COPY *.go ./
COPY static ./static
RUN go build -trimpath -ldflags="-s -w" -o /out/omdb-api-manager .

FROM alpine:3.21
WORKDIR /app
COPY --from=builder /out/omdb-api-manager /app/omdb-api-manager
COPY .env.example /app/.env.example
COPY omdb_keys.example.txt /app/omdb_keys.example.txt
COPY client_keys.example.txt /app/client_keys.example.txt
ENV LISTEN_ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/app/omdb-api-manager"]
