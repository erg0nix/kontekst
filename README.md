# kontekst

A local AI agent daemon with a gRPC interface.

The daemon runs in the background, manages conversation context, and executes tools. LLM providers are configured per-agent. Clients connect over gRPC to interact with it.

## Requirements

- [llama-server](https://github.com/ggerganov/llama.cpp) installed
- A GGUF model in `~/models/`

## Build

```bash
make build
```

## Run

```bash
make run llama start --background  # start llama-server
make run start                     # start daemon
make run ps                        # status
make run "prompt"                  # send a prompt
make run stop                      # stop daemon
make run llama stop                # stop llama-server
```

Configuration, agents, sessions, and history are stored in `~/.kontekst/`.
