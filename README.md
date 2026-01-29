# kontekst

A local AI agent daemon with a gRPC interface.

The daemon runs in the background, handles LLM inference via llama-server, manages conversation context, and executes tools. Clients connect over gRPC to interact with it.

## Requirements

- [llama-server](https://github.com/ggerganov/llama.cpp) installed
- A GGUF model in `~/models/`

## Build

```bash
make build
```

## Run

```bash
make run start        # start daemon
make run stop         # stop daemon
make run ps           # status
make run "prompt"     # send a prompt
```

Configuration, sessions, and history are stored in `~/.kontekst/`.
