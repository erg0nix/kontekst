.PHONY: build run test

RUN_ARGS ?= $(filter-out $@,$(MAKECMDGOALS))

build:
	go build -o bin/kontekst ./cmd/cli

run:
	./bin/kontekst $(RUN_ARGS)

test:
	go test ./...

%:
	@:
