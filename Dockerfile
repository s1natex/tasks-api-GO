ARG GO_VERSION=1.24.3
FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/tasks-api .

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=builder /out/tasks-api /app/tasks-api

EXPOSE 8080 8081
USER nonroot:nonroot
ENTRYPOINT ["/app/tasks-api"]
