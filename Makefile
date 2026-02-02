.PHONY: build proto run test tools

RUN_ARGS ?= $(filter-out $@,$(MAKECMDGOALS))

build: tools proto
	go build -o bin/kontekst-daemon ./cmd/daemon
	go build -o bin/kontekst ./cmd/cli

proto:
	protoc --go_out=./ --go_opt=module=github.com/erg0nix/kontekst \
		--go-grpc_out=./ --go-grpc_opt=module=github.com/erg0nix/kontekst \
		proto/kontekst.proto

tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc

run: build
	./bin/kontekst $(RUN_ARGS)

test:
	go test ./...

%:
	@:
