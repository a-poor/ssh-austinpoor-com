WEBSITE_DIR ?= ../austinpoor-dot-com

.PHONY: content build run serve image test

content:
	go run ./cmd/sync-content -source "$(WEBSITE_DIR)"

build: content
	go build -o bin/ssh-austinpoor-com .

run: content
	go run .

serve: content
	go run . serve

image: content
	docker build -t ssh-austinpoor-com .

test:
	go test ./...
