# Go ≥ 1.21; GOTOOLCHAIN=auto installs the toolchain from go.mod (e.g. 1.25.x) during go mod download / build.
FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
ENV GOTOOLCHAIN=auto
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /server .

EXPOSE 8080

CMD ["./server"]
