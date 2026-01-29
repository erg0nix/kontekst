.PHONY: build proto run

RUN_ARGS ?= $(filter-out $@,$(MAKECMDGOALS))

build: proto
	go build -o bin/kontekst-daemon ./cmd/daemon
	go build -o bin/kontekst ./cmd/cli

proto:
	protoc --go_out=./ --go_opt=module=kontekst-go \
		--go-grpc_out=./ --go-grpc_opt=module=kontekst-go \
		proto/kontekst.proto

run: build
	./bin/kontekst $(RUN_ARGS)

%:
	@:
