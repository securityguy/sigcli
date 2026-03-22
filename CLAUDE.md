# sigcli — Signal CLI Client

## Project Status: UNRELEASED

MIT-licensed. This is a separate work from sigd (AGPL-3.0).

sigcli communicates with sigd over TCP using the JSON-RPC 2.0 protocol.
It contains zero Signal or cryptographic code.

## About

sigcli is both:
1. A CLI tool for interacting with a running sigd daemon
2. Eventually, the home of a Go client library (`pkg/sigcli/`) that other
   applications can import to communicate with sigd

## Commands

- `sigcli link` — link sigd to a Signal account
- `sigcli status` — show connection status
- `sigcli send <to> <message>` — send a message
- `sigcli receive` — poll for pending messages
- `sigcli subscribe` — stream incoming messages (writes to debug.log)

## Building

```bash
make build    # produces bin/sigcli
make install  # installs to ~/.local/bin
```

No CGO, no Rust, no external dependencies — pure Go stdlib.
