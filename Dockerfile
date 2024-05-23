ARG GO_VERSION=1
FROM golang:${GO_VERSION}-bookworm as builder

WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN go build -v -o /server ./cmd/server


FROM debian:bookworm

COPY --from=builder /server /usr/local/bin/
CMD ["server"]
