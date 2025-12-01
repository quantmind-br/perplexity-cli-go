# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Perplexity-go is a Go CLI and library for interacting with the Perplexity AI API. It provides Chrome TLS fingerprint spoofing via `tls-client`, SSE streaming response parsing, and terminal-rendered markdown output.

## Architecture

```
perplexity-go/
├── cmd/perplexity/     # CLI commands (Cobra)
│   ├── main.go         # Entry point
│   ├── root.go         # Main query command + flag handling
│   ├── config.go       # Interactive config menu (uses huh TUI)
│   ├── cookies.go      # Cookie management (import/status/clear/path)
│   ├── history.go      # Query history
│   └── version.go      # Version info
├── pkg/
│   ├── client/         # API client (exported)
│   │   ├── client.go   # Main client struct + Search methods
│   │   ├── http.go     # TLS-client wrapper with Chrome fingerprint
│   │   ├── search.go   # SSE parsing, payload building
│   │   └── upload.go   # S3 file upload
│   └── models/         # Data types (exported)
│       ├── types.go    # Mode, Model, Source enums
│       ├── request.go  # SearchRequest, SearchOptions
│       └── response.go # SearchResponse, StreamChunk, blocks
└── internal/
    ├── auth/           # Cookie loading (JSON/Netscape formats)
    ├── config/         # Viper-based config (~/.perplexity-cli/config.json)
    ├── history/        # JSONL history writer
    └── ui/             # Glamour/Lipgloss rendering + interactive config menu
```

### Key Dependencies
- `github.com/bogdanfinn/tls-client` + `fhttp`: Chrome TLS fingerprint impersonation
- `github.com/spf13/cobra` + `viper`: CLI framework and config
- `github.com/charmbracelet/glamour` + `lipgloss`: Terminal markdown rendering
- `github.com/charmbracelet/huh`: Interactive terminal UI forms

## Common Commands

```bash
# Build
make build                    # Build to ./build/perplexity
make build-release            # Optimized build with -s -w -trimpath

# Install
make install                  # System install (/usr/local/bin, requires sudo)
make install-user             # User install (~/.local/bin)

# Test
make test                     # Run all tests
make test-coverage            # With coverage summary
make test-coverage-html       # Generate HTML coverage report
go test ./pkg/client/... -v   # Run specific package tests
go test ./... -run TestName   # Run specific test

# Run directly
make run ARGS='"What is Go?"'
./build/perplexity "query" --model gpt51 --mode pro --stream
```

## API Implementation Details

### SSE Response Parsing (`pkg/client/search.go`)
- Delimiter: `\r\n\r\n` (also handles `\n\n`)
- Format: `event: message\r\ndata: {...json...}`
- Double JSON parsing: outer JSON has `text` field containing inner JSON with `blocks`
- Blocks: `markdown_block` (answer + citations), `web_search_results`

### Mode Mapping
| CLI Mode       | API mode   | model_preference |
|----------------|------------|------------------|
| fast           | concise    | turbo           |
| pro/default    | copilot    | (from model)    |
| reasoning      | copilot    | + is_pro_reasoning_mode=true |
| deep-research  | copilot    | pplx_alpha      |

**Special case:** `gpt51_thinking` model forces `concise` + `turbo` regardless of mode.

### Available Models

**Pro Mode:**
`pplx_pro`, `gpt51`, `grok41nonreasoning`, `experimental`, `claude45sonnet`

**Reasoning Mode:**
`gemini30pro`, `gpt51_thinking`, `grok41reasoning`, `kimik2thinking`, `claude45sonnetthinking`

### Configuration
- Config file: `~/.perplexity-cli/config.json`
- Cookie file: `~/.perplexity-cli/cookies.json`
- Environment prefix: `PERPLEXITY_` (e.g., `PERPLEXITY_DEFAULT_MODEL`)
- Language format: `xx-XX` (e.g., `en-US`, `pt-BR`)

## Cookie Authentication

Cookies must be exported from browser (JSON format via extension or Netscape format). Required cookie: `next-auth.csrf-token`.

```bash
perplexity import-cookies cookies.json  # Import cookies
perplexity cookies status               # Check authentication
perplexity cookies clear                # Clear saved cookies
perplexity cookies path                 # Show cookie file path
```

## Interactive Configuration

The `perplexity config` command opens an interactive menu to manage settings:

```bash
perplexity config                       # Open interactive config menu
perplexity config path                  # Show config file path
```

Configuration options available in the interactive menu:
- Model: Default AI model
- Mode: Search mode (fast, pro, reasoning, deep-research, default)
- Language: Response language
- Sources: Search sources (web, scholar, social)
- Streaming: Enable/disable streaming output
- Incognito: Enable/disable history saving

## Testing Notes

- Tests use table-driven patterns
- HTTP tests mock the tls-client
- Run `go test -race ./...` to check for race conditions
- Coverage target: 80% for new code
