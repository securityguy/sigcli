# sigcli

> [!WARNING]
> **This project is a work in progress and should not be used in production.**

A CLI client and Go library for interacting with [sigd](https://github.com/securityguy/sigd), a Signal Messenger daemon that exposes a [JSON-RPC 2.0](https://www.jsonrpc.org/specification) TCP interface.

sigcli implements the sigd JSON-RPC 2.0 protocol. It contains zero Signal or cryptographic code — pure Go stdlib, no CGO, no external dependencies. For the daemon that handles Signal connectivity, see the separate project [sigd](https://github.com/securityguy/sigd).

```
[sigd daemon] <--tcp/json-rpc--> [sigcli]
```

## Requirements

- Go 1.22+
- A running [sigd](https://github.com/securityguy/sigd) instance

## Building

```bash
make build    # produces bin/sigcli
make install  # installs to ~/.local/bin
```

No CGO, no Rust, no external dependencies — pure Go stdlib.

## Usage

```bash
sigcli [--addr 127.0.0.1:7777] <command> [args]

Commands:
  link              Link sigd to a Signal account (shows QR, waits for completion)
  status            Show current link and connection status
  send <to> <msg>   Send a text message
  subscribe         Subscribe to incoming messages, print to stdout and append to debug.log
  receive           Poll once for pending messages and print them
```

## Example: first-time setup and testing

This walkthrough assumes sigd is installed and your Signal account is not yet linked.

**Terminal 1 — start sigd**
```bash
sigd --db sigd.db --log-level debug
```

**Terminal 2 — link to your Signal account**
```bash
sigcli link
```
sigd prints a QR code in Terminal 1. On your phone: **Settings → Linked Devices → Link New Device** → scan the QR code. sigcli exits once linking completes.

**Verify the link**
```bash
sigcli status
```

**Send a message** (use a full E.164 phone number)
```bash
sigcli send +12345551234 "hello from sigd"
```

**Receive pending messages**
```bash
sigcli receive
```

**Stream incoming messages** (blocks; prints to stdout as messages arrive)
```bash
sigcli subscribe
```

sigd maintains its WebSocket connection to Signal servers continuously — `subscribe` connects your terminal to sigd's push stream. If you disconnect, incoming messages are queued in sigd's database and available via `receive`.

To start fresh (unlink and wipe state):
```bash
rm sigd.db sigd.db-wal sigd.db-shm 2>/dev/null; sigd --db sigd.db
```

## Not affiliated with Signal

This project is not affiliated with, endorsed by, or associated with Signal Messenger, LLC or the Signal Technology Foundation in any way. Signal® is a registered trademark of Signal Messenger, LLC. The Signal name and logo are the property of Signal Messenger, LLC and are used here solely for identification purposes.

## Copyright and License

Copyright (c) 2026 Tenebris Technologies Inc.

This software is licensed under the [MIT License](LICENSE). Please see `LICENSE` for details.

## Trademarks

Any trademarks referenced are the property of their respective owners, used for identification only, and do not imply sponsorship, endorsement, or affiliation.

## No Warranty

**(zilch, none, void, nil, null, "", {}, 0x00, 0b00000000, EOF)**

THIS SOFTWARE IS PROVIDED "AS IS," WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE, AND NON-INFRINGEMENT. IN NO EVENT SHALL THE COPYRIGHT HOLDERS OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

Made in Canada with internationally sourced components.
