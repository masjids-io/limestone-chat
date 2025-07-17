FROM golang:1.24.3-alpine as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0
RUN go build -o /bin/app -ldflags="-s -w" cmd/main.go

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /bin/app /app/app
COPY --from=builder /app/.env /app/.env

USER nonroot

ENTRYPOINT ["/app/app"]