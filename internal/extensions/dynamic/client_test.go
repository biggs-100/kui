package dynamic

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"testing"
)

// buildMockServer writes a Go source file, compiles it, and returns the binary path.
// The source must be a main package that reads JSON-RPC from stdin and writes to stdout.
func buildMockServer(t *testing.T, name, src string) string {
	t.Helper()
	tmpDir := t.TempDir()
	srcPath := tmpDir + "/" + name + ".go"
	binPath := tmpDir + "/" + name
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	build := exec.Command("go", "build", "-o", binPath, srcPath)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v\n%s", name, err, out)
	}
	return binPath
}

// newTestClient starts a binary and wraps it in a Client.
func newTestClient(t *testing.T, binPath string) (*Client, *exec.Cmd) {
	t.Helper()
	cmd := exec.Command(binPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cmd.Process.Kill() })
	c := &Client{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdoutPipe,
		scanner: bufio.NewScanner(stdoutPipe),
	}
	return c, cmd
}

// --- Mock server sources ---
// Uses []byte concatenation to embed backticks in struct tags.

var echoServerSrc = []byte(`package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type req struct {
	JSONRPC string ` + "`" + `json:"jsonrpc"` + "`" + `
	ID      int    ` + "`" + `json:"id"` + "`" + `
	Method  string ` + "`" + `json:"method"` + "`" + `
}

type resp struct {
	JSONRPC string      ` + "`" + `json:"jsonrpc"` + "`" + `
	ID      int         ` + "`" + `json:"id"` + "`" + `
	Result  interface{} ` + "`" + `json:"result"` + "`" + `
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var r req
		if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
			continue
		}
		res, _ := json.Marshal(resp{JSONRPC: "2.0", ID: r.ID, Result: map[string]string{"method": r.Method}})
		res = append(res, '\n')
		fmt.Fprint(os.Stdout, string(res))
	}
}
`)

var versionMismatchServerSrc = []byte(`package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type req struct {
	JSONRPC string          ` + "`" + `json:"jsonrpc"` + "`" + `
	ID      int             ` + "`" + `json:"id"` + "`" + `
	Method  string          ` + "`" + `json:"method"` + "`" + `
	Params  json.RawMessage ` + "`" + `json:"params"` + "`" + `
}

type resp struct {
	JSONRPC string      ` + "`" + `json:"jsonrpc"` + "`" + `
	ID      int         ` + "`" + `json:"id"` + "`" + `
	Result  interface{} ` + "`" + `json:"result"` + "`" + `
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var r req
		if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
			continue
		}
		if r.Method == "initialize" {
			res, _ := json.Marshal(resp{JSONRPC: "2.0", ID: r.ID, Result: map[string]interface{}{
				"protocolVersion": "wrong/1",
			}})
			res = append(res, '\n')
			fmt.Fprint(os.Stdout, string(res))
		}
	}
}
`)

var listToolsServerSrc = []byte(`package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type req struct {
	JSONRPC string          ` + "`" + `json:"jsonrpc"` + "`" + `
	ID      int             ` + "`" + `json:"id"` + "`" + `
	Method  string          ` + "`" + `json:"method"` + "`" + `
	Params  json.RawMessage ` + "`" + `json:"params"` + "`" + `
}

type resp struct {
	JSONRPC string      ` + "`" + `json:"jsonrpc"` + "`" + `
	ID      int         ` + "`" + `json:"id"` + "`" + `
	Result  interface{} ` + "`" + `json:"result"` + "`" + `
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var r req
		if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
			continue
		}
		var res []byte
		switch r.Method {
		case "initialize":
			res, _ = json.Marshal(resp{JSONRPC: "2.0", ID: r.ID, Result: map[string]interface{}{
				"protocolVersion": "kui-ext/1",
			}})
		case "extensions/list":
			res, _ = json.Marshal(resp{JSONRPC: "2.0", ID: r.ID, Result: map[string]interface{}{
				"tools": []map[string]interface{}{
					{"name": "foo", "description": "foo desc", "inputSchema": map[string]interface{}{"type": "object"}},
					{"name": "bar", "description": "bar desc", "inputSchema": map[string]interface{}{"type": "object"}},
				},
			}})
		}
		if res != nil {
			res = append(res, '\n')
			fmt.Fprint(os.Stdout, string(res))
		}
	}
}
`)

var callToolServerSrc = []byte(`package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type req struct {
	JSONRPC string          ` + "`" + `json:"jsonrpc"` + "`" + `
	ID      int             ` + "`" + `json:"id"` + "`" + `
	Method  string          ` + "`" + `json:"method"` + "`" + `
	Params  json.RawMessage ` + "`" + `json:"params"` + "`" + `
}

type resp struct {
	JSONRPC string      ` + "`" + `json:"jsonrpc"` + "`" + `
	ID      int         ` + "`" + `json:"id"` + "`" + `
	Result  interface{} ` + "`" + `json:"result"` + "`" + `
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var r req
		if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
			continue
		}
		var res []byte
		switch r.Method {
		case "initialize":
			res, _ = json.Marshal(resp{JSONRPC: "2.0", ID: r.ID, Result: map[string]interface{}{
				"protocolVersion": "kui-ext/1",
			}})
		case "extensions/call":
			var params struct {
				Name      string          ` + "`" + `json:"name"` + "`" + `
				Arguments json.RawMessage ` + "`" + `json:"arguments"` + "`" + `
			}
			json.Unmarshal(r.Params, &params)
			res, _ = json.Marshal(resp{JSONRPC: "2.0", ID: r.ID, Result: map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": "result:" + params.Name},
				},
			}})
		}
		if res != nil {
			res = append(res, '\n')
			fmt.Fprint(os.Stdout, string(res))
		}
	}
}
`)

var errorToolServerSrc = []byte(`package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type req struct {
	JSONRPC string          ` + "`" + `json:"jsonrpc"` + "`" + `
	ID      int             ` + "`" + `json:"id"` + "`" + `
	Method  string          ` + "`" + `json:"method"` + "`" + `
	Params  json.RawMessage ` + "`" + `json:"params"` + "`" + `
}

type resp struct {
	JSONRPC string      ` + "`" + `json:"jsonrpc"` + "`" + `
	ID      int         ` + "`" + `json:"id"` + "`" + `
	Result  interface{} ` + "`" + `json:"result"` + "`" + `
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var r req
		if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
			continue
		}
		var res []byte
		switch r.Method {
		case "initialize":
			res, _ = json.Marshal(resp{JSONRPC: "2.0", ID: r.ID, Result: map[string]interface{}{
				"protocolVersion": "kui-ext/1",
			}})
		case "extensions/call":
			res, _ = json.Marshal(resp{JSONRPC: "2.0", ID: r.ID, Result: map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": "tool failed"},
				},
				"isError": true,
			}})
		}
		if res != nil {
			res = append(res, '\n')
			fmt.Fprint(os.Stdout, string(res))
		}
	}
}
`)

// --- Tests ---

func TestSendRequestSerialization(t *testing.T) {
	binPath := buildMockServer(t, "echo", string(echoServerSrc))
	c, _ := newTestClient(t, binPath)

	ctx := context.Background()
	var result map[string]string
	if err := c.sendRequest(ctx, "test/method", nil, &result); err != nil {
		t.Fatalf("sendRequest error: %v", err)
	}
	if result["method"] != "test/method" {
		t.Errorf("got method %q, want %q", result["method"], "test/method")
	}
}

func TestClientVersionMismatch(t *testing.T) {
	binPath := buildMockServer(t, "badver", string(versionMismatchServerSrc))
	c, _ := newTestClient(t, binPath)

	ctx := context.Background()
	err := c.Initialize(ctx)
	if err == nil {
		t.Fatal("expected error for version mismatch, got nil")
	}

	var protoErr *ProtocolError
	if !errors.As(err, &protoErr) {
		t.Fatalf("expected ProtocolError, got: %v", err)
	}
}

func TestClientListTools(t *testing.T) {
	binPath := buildMockServer(t, "listserver", string(listToolsServerSrc))
	c, _ := newTestClient(t, binPath)

	ctx := context.Background()
	if err := c.Initialize(ctx); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	tools, err := c.ListTools(ctx)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("got %d tools, want 2", len(tools))
	}
	if tools[0].Name != "foo" {
		t.Errorf("tools[0].Name = %q, want %q", tools[0].Name, "foo")
	}
	if tools[1].Name != "bar" {
		t.Errorf("tools[1].Name = %q, want %q", tools[1].Name, "bar")
	}
}

func TestClientCallTool(t *testing.T) {
	binPath := buildMockServer(t, "callserver", string(callToolServerSrc))
	c, _ := newTestClient(t, binPath)

	ctx := context.Background()
	if err := c.Initialize(ctx); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	text, err := c.CallTool(ctx, "mytool", json.RawMessage(`{"arg":"val"}`))
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if text != "result:mytool" {
		t.Errorf("got %q, want %q", text, "result:mytool")
	}
}

func TestClientCallToolError(t *testing.T) {
	binPath := buildMockServer(t, "errserver", string(errorToolServerSrc))
	c, _ := newTestClient(t, binPath)

	ctx := context.Background()
	if err := c.Initialize(ctx); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	_, err := c.CallTool(ctx, "failtool", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error for isError response, got nil")
	}
}

func TestClientClose(t *testing.T) {
	binPath := buildMockServer(t, "echo2", string(echoServerSrc))
	c, _ := newTestClient(t, binPath)

	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	// Double close should be safe.
	if err := c.Close(); err != nil {
		t.Fatalf("double Close: %v", err)
	}
}

func TestNewClientEmptyEntryPoint(t *testing.T) {
	ctx := context.Background()
	_, err := NewClient(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty entry point, got nil")
	}
}
