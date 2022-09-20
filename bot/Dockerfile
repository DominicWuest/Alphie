FROM golang:1.18-alpine3.15 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /Alphie

# -------------------------------- #

FROM alpine:latest AS deployment
WORKDIR /usr/src/app
RUN apk add tzdata
COPY --from=builder /Alphie .
CMD ./Alphie