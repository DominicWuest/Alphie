FROM golang:1.18-alpine3.15 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /server

# -------------------------------- #

FROM alpine:latest AS deployment
WORKDIR /usr/src/app
RUN apk add tzdata
COPY --from=builder /server .
CMD ./server