FROM golang:1.18
WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
CMD ["go", "run", "Alphie.go"]

# TODO: Builder + Production for Kubernetes