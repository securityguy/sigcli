# sigcli — TODO

## Up Next

- [ ] **sigcli library** — extract JSON-RPC client logic into `pkg/sigcli/` as an importable Go package
  - Clean API: `client.Send()`, `client.Subscribe()`, `client.Link()`, etc.
  - Handles reconnection, message parsing, error handling
  - Update `cmd/sigcli/main.go` to use the library instead of inline JSON-RPC code

## Features

- [ ] **Pretty output** — formatted tables/colors for interactive use (optional, keep plain text as default)
- [ ] **Config file** — `~/.config/sigcli/config.toml` for default `--addr`, etc.
- [ ] **Shell completion** — bash/zsh completion scripts
- [ ] **Pipe-friendly** — ensure all output works well in scripts (exit codes, machine-readable mode)
