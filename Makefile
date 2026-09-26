ENV ?= dev

.PHONY: up converge smoke down test

up:
	go run ./cmd/platform up -env $(ENV)

converge:
	go run ./cmd/platform converge -env $(ENV)

smoke:
	go run ./cmd/platform smoke -env $(ENV)

down:
	go run ./cmd/platform down -env $(ENV)

test:
	go test ./...
