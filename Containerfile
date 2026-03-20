FROM golang:1.22-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -o /rulegen ./cmd/rulegen/

FROM alpine:3.20

RUN apk add --no-cache ca-certificates
COPY --from=builder /rulegen /usr/local/bin/rulegen

EXPOSE 8080
ENTRYPOINT ["rulegen", "serve"]
