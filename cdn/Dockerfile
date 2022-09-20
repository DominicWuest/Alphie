FROM golang:1.18-alpine3.15 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY server.go .

RUN go build -o /server

# -------------------------------- #

FROM alpine:latest AS deployment
WORKDIR /usr/src/app
RUN apk add tzdata ffmpeg
COPY --from=builder /server .
EXPOSE 1337
CMD ./server