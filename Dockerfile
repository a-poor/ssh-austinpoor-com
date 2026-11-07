ARG GO_VERSION=1.27.3
FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
# Fail early if content has not been imported before sending the build context.
COPY content/posts.json ./content/posts.json
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/ssh-austinpoor-com .


FROM debian:bookworm-slim

WORKDIR /app
ENV HOST=0.0.0.0 PORT=2222 HOST_KEY_PATH=/shared/.ssh/id_ed25519
COPY --from=builder /out/ssh-austinpoor-com /usr/local/bin/ssh-austinpoor-com
EXPOSE 2222
ENTRYPOINT ["/usr/local/bin/ssh-austinpoor-com"]
CMD ["serve"]
