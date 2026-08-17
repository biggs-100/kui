package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// fakeMCPServerSrc is the source code for a minimal MCP server that handles
// initialize, tools/list, and tools/call. It reads JSON-RPC from stdin and
// writes responses to stdout. This is compiled during test setup.
const fakeMCPServerSrc = `package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req struct {
			ID     int    ` + "`" + `json:"id"` + "`" + `
			Method string ` + "`" + `json:"method"` + "`" + `
		}
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}
		if req.ID == 0 {
			continue
		}
		var resp string
		switch req.Method {
		case "initialize":
			resp = fmt.Sprintf(` + "`" + `{"jsonrpc":"2.0","id":%d,"result":{"protocolVersion":"2025-03-26","capabilities":{},"serverInfo":{"name":"fake-mcp","version":"0.1.0"}}}` + "`" + `, req.ID)
		case "tools/list":
			resp = fmt.Sprintf(` + "`" + `{"jsonrpc":"2.0","id":%d,"result":{"tools":[{"name":"echo","description":"Echoes back the input","inputSchema":{"type":"object","properties":{}}}]}}` + "`" + `, req.ID)
		case "tools/call":
			resp = fmt.Sprintf(` + "`" + `{"jsonrpc":"2.0","id":%d,"result":{"content":[{"type":"text","text":"echo result"}],"isError":false}}` + "`" + `, req.ID)
		default:
			resp = fmt.Sprintf(` + "`" + `{"jsonrpc":"2.0","id":%d,"result":{}}` + "`" + `, req.ID)
		}
		fmt.Fprintln(os.Stdout, resp)
		os.Stdout.Sync()
	}
}
`

// testingTB is the minimal interface satisfied by *testing.T and *testing.B.
type testingTB interface {
	Helper()
	TempDir() string
	Fatalf(string, ...interface{})
}

// compileFakeMCPServer builds a minimal MCP server binary and returns its path.
func compileFakeMCPServer(tb testingTB) string {
	tb.Helper()
	dir := tb.TempDir()

	srcPath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(srcPath, []byte(fakeMCPServerSrc), 0o644); err != nil {
		tb.Fatalf("write fake mcp server source: %v", err)
	}

	name := "fake-mcp-server"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binPath := filepath.Join(dir, name)

	cmd := exec.Command("go", "build", "-o", binPath, srcPath)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		tb.Fatalf("compile fake mcp server: %v\n%s", err, out)
	}

	return binPath
}
