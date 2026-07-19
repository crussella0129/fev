#Animus Fev

A local-first agentic CLI harness. Your personal assistant for navigating the digital space.

- **Model-agnostic** -- works with any OpenAI-compatible API (Ollama, llama.cpp, vLLM, LM Studio)
- **Single binary** -- `go install` or download. No Python, no Node.js, no Docker.
- **Ecosystem-compatible** -- reads CLAUDE.md, speaks MCP, uses familiar tool names

## Quick Start

```bash
go install github.com/crussella0129/fev/cmd/fev@latest
fev
```

## Build from Source

```bash
git clone https://github.com/crussella0129/fev.git
cd fev
make build
./bin/fev version
```

## License

MIT
