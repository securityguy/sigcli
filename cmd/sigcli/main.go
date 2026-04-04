// Command sigcli is a JSON-RPC 2.0 test client for the sigd daemon.
// It communicates with sigd over TCP using stdlib only — no CGO, no external dependencies.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mdp/qrterminal/v3"
	"github.com/securityguy/sigcli/global"
)

const defaultAddr = "127.0.0.1:7777"

// ---------------------------------------------------------------------------
// JSON-RPC 2.0 types
// ---------------------------------------------------------------------------

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// notification is a JSON-RPC 2.0 server-push message with no id field.
type notification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// ---------------------------------------------------------------------------
// rpcClient
// ---------------------------------------------------------------------------

type rpcClient struct {
	conn    net.Conn
	scanner *bufio.Scanner
	writer  *bufio.Writer
	id      int
}

// dial connects to sigd at addr and returns an rpcClient.
func dial(addr string) (*rpcClient, error) {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to sigd at %s — is it running?", addr)
	}
	return &rpcClient{
		conn:    conn,
		scanner: bufio.NewScanner(conn),
		writer:  bufio.NewWriter(conn),
	}, nil
}

// call sends a JSON-RPC 2.0 request and returns the raw result.
func (c *rpcClient) call(method string, params any) (json.RawMessage, error) {
	c.id++
	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      c.id,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	data = append(data, '\n')

	if _, err := c.writer.Write(data); err != nil {
		return nil, fmt.Errorf("write request: %w", err)
	}
	if err := c.writer.Flush(); err != nil {
		return nil, fmt.Errorf("flush request: %w", err)
	}

	if !c.scanner.Scan() {
		if err := c.scanner.Err(); err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}
		return nil, fmt.Errorf("connection closed by sigd")
	}

	var resp rpcResponse
	if err := json.Unmarshal(c.scanner.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("%s (code: %d)", resp.Error.Message, resp.Error.Code)
	}

	return resp.Result, nil
}

// close closes the underlying connection.
func (c *rpcClient) close() {
	_ = c.conn.Close()
}

// ---------------------------------------------------------------------------
// Commands
// ---------------------------------------------------------------------------

// linkReply matches the schema.LinkReply fields returned by link.request and link.status.
type linkReply struct {
	Status string `json:"status"`
	URI    string `json:"uri,omitempty"`
	ACI    string `json:"aci,omitempty"`
	Phone  string `json:"phone,omitempty"`
	Error  string `json:"error,omitempty"`
}

// cmdLink starts the device linking flow using link.request / link.status.
func cmdLink(addr string) error {
	c, err := dial(addr)
	if err != nil {
		return err
	}
	defer c.close()

	// Step 1: call "link.request" to start the flow and get the QR URI.
	result, err := c.call("link.request", map[string]string{"name": "sigcli"})
	if err != nil {
		return err
	}

	var reply linkReply
	if err := json.Unmarshal(result, &reply); err != nil {
		return fmt.Errorf("parse link.request result: %w", err)
	}

	switch reply.Status {
	case "complete":
		fmt.Printf("Linked! ACI: %s  Phone: %s\n", reply.ACI, reply.Phone)
		return nil
	case "error":
		return fmt.Errorf("link.request failed: %s", reply.Error)
	case "pending":
		// Expected path — display QR and poll.
	default:
		return fmt.Errorf("unexpected link.request status: %s", reply.Status)
	}

	if reply.URI != "" {
		qrterminal.GenerateWithConfig(reply.URI, qrterminal.Config{
			Level:          qrterminal.L,
			Writer:         os.Stdout,
			HalfBlocks:     true,
			BlackChar:      "\033[40m \033[0m",     // both black
			WhiteChar:      "\033[47m \033[0m",     // both white
			BlackWhiteChar: "\033[40;37m▄\033[0m", // top black, bottom white
			WhiteBlackChar: "\033[47;30m▄\033[0m", // top white, bottom black
			QuietZone:      1,
		})
		fmt.Println("Scan the QR code above, or use this URI:")
		fmt.Println(reply.URI)
		fmt.Println()
	}
	fmt.Printf("Waiting for device scan (up to %v)...\n", global.LinkWaitTimeout)

	// Step 2: poll link.status every 2 seconds until complete or timeout.
	deadline := time.Now().Add(global.LinkWaitTimeout)
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)

		statusResult, err := c.call("link.status", struct{}{})
		if err != nil {
			return fmt.Errorf("link.status: %w", err)
		}

		var statusReply linkReply
		if err := json.Unmarshal(statusResult, &statusReply); err != nil {
			return fmt.Errorf("parse link.status result: %w", err)
		}

		switch statusReply.Status {
		case "pending":
			// Keep polling.
			continue
		case "complete":
			fmt.Printf("Linked! ACI: %s  Phone: %s\n", statusReply.ACI, statusReply.Phone)
			return nil
		case "error":
			return fmt.Errorf("linking failed: %s", statusReply.Error)
		default:
			return fmt.Errorf("unexpected link.status status: %s", statusReply.Status)
		}
	}

	return fmt.Errorf("timed out waiting for device scan after %v", global.LinkWaitTimeout)
}

// cmdStatus prints the current link and connection status.
func cmdStatus(addr string) error {
	c, err := dial(addr)
	if err != nil {
		return err
	}
	defer c.close()

	result, err := c.call("status", struct{}{})
	if err != nil {
		return err
	}

	var status struct {
		Linked    bool   `json:"linked"`
		Connected bool   `json:"connected"`
		ACI       string `json:"aci"`
	}
	if err := json.Unmarshal(result, &status); err != nil {
		return fmt.Errorf("parse status result: %w", err)
	}

	linked := "no"
	if status.Linked {
		linked = "yes"
	}
	connected := "no"
	if status.Connected {
		connected = "yes"
	}

	fmt.Printf("Linked:    %s\n", linked)
	fmt.Printf("Connected: %s\n", connected)
	if status.ACI != "" {
		fmt.Printf("ACI:       %s\n", status.ACI)
	}
	return nil
}

// cmdSend sends a text message.
func cmdSend(addr, to, body string) error {
	c, err := dial(addr)
	if err != nil {
		return err
	}
	defer c.close()

	result, err := c.call("send", map[string]string{"to": to, "body": body})
	if err != nil {
		return err
	}

	var sent struct {
		Timestamp uint64 `json:"timestamp"`
	}
	if err := json.Unmarshal(result, &sent); err != nil {
		return fmt.Errorf("parse send result: %w", err)
	}

	fmt.Printf("Sent (timestamp: %d)\n", sent.Timestamp)
	return nil
}

// cmdSubscribe subscribes to incoming notifications and streams them to stdout
// and debug.log until SIGINT.
func cmdSubscribe(addr string, raw bool) error {
	c, err := dial(addr)
	if err != nil {
		return err
	}
	defer c.close()

	// Subscribe.
	result, err := c.call("subscribe", struct{}{})
	if err != nil {
		return err
	}

	var ack struct {
		Subscribed bool `json:"subscribed"`
	}
	if err := json.Unmarshal(result, &ack); err != nil || !ack.Subscribed {
		return fmt.Errorf("unexpected subscribe response")
	}

	// Open (or create) debug.log for appending.
	logFile, err := os.OpenFile("debug.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("open debug.log: %w", err)
	}
	defer func() {
		_ = logFile.Sync()
		_ = logFile.Close()
	}()

	logWriter := bufio.NewWriter(logFile)

	// Handle Ctrl+C: flush and exit cleanly.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		_ = logWriter.Flush()
		_ = logFile.Sync()
		_ = logFile.Close()
		c.close()
		os.Exit(0)
	}()

	fmt.Println("Subscribed. Waiting for messages (Ctrl+C to stop)...")

	// Enter notification read loop. After subscribe the server sends raw
	// notifications — no more request/response framing.
	scanner := c.scanner
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var n notification
		if err := json.Unmarshal(line, &n); err != nil {
			continue
		}

		ts := time.Now().Format("2006-01-02 15:04:05")
		var output string

		if raw {
			var pretty bytes.Buffer
			if err := json.Indent(&pretty, line, "", "  "); err == nil {
				output = fmt.Sprintf("[%s] %s", ts, pretty.String())
			} else {
				output = fmt.Sprintf("[%s] %s", ts, string(line))
			}
			fmt.Println(output)
			if _, err := fmt.Fprintln(logWriter, output); err == nil {
				_ = logWriter.Flush()
			}
			continue
		}

		switch n.Method {
		case "message":
			var p struct {
				From         struct{ ID string `json:"id"` } `json:"from"`
				To           struct{ ID string `json:"id"` } `json:"to"`
				Body         string `json:"body"`
				Timestamp    uint64 `json:"timestamp"`
				Type         string `json:"type"`
				RefTimestamp uint64 `json:"ref_timestamp"`
			}
			if err := json.Unmarshal(n.Params, &p); err == nil {
				peer := p.From.ID
				dir := "FROM"
				if peer == "" {
					peer = p.To.ID
					dir = "TO"
				}
				switch p.Type {
				case "delete":
					output = fmt.Sprintf("[%s] %d DELETE [%d] %s %s", ts, p.Timestamp, p.RefTimestamp, dir, peer)
				case "edit":
					output = fmt.Sprintf("[%s] %d EDIT [%d] %s %s: %s", ts, p.Timestamp, p.RefTimestamp, dir, peer, p.Body)
				default:
					output = fmt.Sprintf("[%s] %d TEXT %s %s: %s", ts, p.Timestamp, dir, peer, p.Body)
				}
			}

		case "receipt":
			var p struct {
				From struct{ ID string `json:"id"` } `json:"from"`
				Type string                          `json:"type"`
			}
			if err := json.Unmarshal(n.Params, &p); err == nil {
				output = fmt.Sprintf("[%s] RECEIPT from %s: %s", ts, p.From.ID, p.Type)
			}

		case "status":
			var p struct {
				Connected bool `json:"connected"`
			}
			if err := json.Unmarshal(n.Params, &p); err == nil {
				connStr := "disconnected"
				if p.Connected {
					connStr = "connected"
				}
				output = fmt.Sprintf("[%s] STATUS: %s", ts, connStr)
			}

		default:
			output = fmt.Sprintf("[%s] NOTIFICATION %s: %s", ts, n.Method, string(n.Params))
		}

		if output == "" {
			continue
		}

		fmt.Println(output)

		if _, err := fmt.Fprintln(logWriter, output); err == nil {
			_ = logWriter.Flush()
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read error: %w", err)
	}

	return nil
}

// cmdReceive polls once for pending messages.
func cmdReceive(addr string, raw bool) error {
	c, err := dial(addr)
	if err != nil {
		return err
	}
	defer c.close()

	result, err := c.call("receive", map[string]int{"timeout": 10})
	if err != nil {
		return err
	}

	if raw {
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, result, "", "  "); err == nil {
			fmt.Println(pretty.String())
		} else {
			fmt.Println(string(result))
		}
		return nil
	}

	var received struct {
		Messages []struct {
			From      string `json:"from"`
			Body      string `json:"body"`
			Timestamp uint64 `json:"timestamp"`
			Type      string `json:"type"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(result, &received); err != nil {
		return fmt.Errorf("parse receive result: %w", err)
	}

	if len(received.Messages) == 0 {
		fmt.Println("No pending messages.")
		return nil
	}

	for _, m := range received.Messages {
		fmt.Printf("[%d] FROM %s: %s\n", m.Timestamp, m.From, m.Body)
	}
	return nil
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

func main() {
	addr := flag.String("addr", defaultAddr, "sigd address")
	jsonOutput := flag.Bool("json", false, "Print raw pretty-printed JSON instead of formatted output (subscribe and receive)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: sigcli [--addr %s] [-json] <command> [args]\n\n", defaultAddr)
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  link              Link sigd to a Signal account (shows QR, waits for completion)\n")
		fmt.Fprintf(os.Stderr, "  status            Show current link and connection status\n")
		fmt.Fprintf(os.Stderr, "  send <to> <msg>   Send a text message\n")
		fmt.Fprintf(os.Stderr, "  subscribe         Subscribe to incoming messages, print to stdout and append to debug.log\n")
		fmt.Fprintf(os.Stderr, "  receive           Poll once for pending messages and print them\n")
	}
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	var err error
	switch args[0] {
	case "link":
		err = cmdLink(*addr)
	case "status":
		err = cmdStatus(*addr)
	case "send":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "Error: send requires <to> and <message> arguments")
			fmt.Fprintln(os.Stderr, "Usage: sigcli send +12345678901 \"Hello world\"")
			os.Exit(1)
		}
		err = cmdSend(*addr, args[1], args[2])
	case "subscribe":
		err = cmdSubscribe(*addr, *jsonOutput)
	case "receive":
		err = cmdReceive(*addr, *jsonOutput)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command %q\n\n", args[0])
		flag.Usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
