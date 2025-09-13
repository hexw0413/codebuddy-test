FROM golang:1.22-alpine AS builder
WORKDIR /app

# Enable Go modules
ENV CGO_ENABLED=0

COPY go.mod ./
RUN go mod download

COPY . .
RUN go build -o /bin/api ./cmd/api

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /home/nonroot
COPY --from=builder /bin/api /bin/api
ENV API_PORT=8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/bin/api"]
