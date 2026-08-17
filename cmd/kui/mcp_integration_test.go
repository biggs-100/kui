package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// MCP Integration Tests — Slice C (REQ-TOOLS-4 MCP tools included)
// ---------------------------------------------------------------------------

// writeMCPConfig writes an mcp.yaml with the given server command to the home dir.
func writeMCPConfig(t *testing.T, home, serverName string, command []string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("servers:\n")
	b.WriteString("  " + serverName + ":\n")
	b.WriteString("    type: local\n")
	b.WriteString("    command: [")
	for i, c := range command {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(`"` + filepath.ToSlash(c) + `"`)
	}
	b.WriteString("]\n")
	if err := os.WriteFile(filepath.Join(home, "mcp.yaml"), []byte(b.String()), 0o644); err != nil {
		t.Fatalf("write mcp.yaml: %v", err)
	}
}

// extractToolNames extracts tool names from the OpenAI function-calling format.
// Each tool is: {"type":"function","function":{"name":"...",...}}
func extractToolNames(tools []map[string]interface{}) []string {
	var names []string
	for _, tool := range tools {
		if fn, ok := tool["function"].(map[string]interface{}); ok {
			if name, ok := fn["name"].(string); ok {
				names = append(names, name)
			}
		}
		// Also check top-level "name" for non-function-calling formats.
		if name, ok := tool["name"].(string); ok {
			names = append(names, name)
		}
	}
	return names
}

// TestCLIMCPNoConfigSuccess covers the case where no mcp.yaml exists:
// kui must still run successfully without MCP tools.
func TestCLIMCPNoConfigSuccess(t *testing.T) {
	var capturedTools []map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Tools []map[string]interface{} `json:"tools"`
		}
		if err := json.Unmarshal(body, &req); err == nil {
			capturedTools = req.Tools
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"noconfig answer"}}]}`)
	}))
	defer srv.Close()

	home := t.TempDir()
	// No mcp.yaml — MCP should not add any tools.

	stdout, stderr, code := runCLI(t, map[string]string{
		"OPENAI_API_KEY":  "sk-test-123",
		"OPENAI_BASE_URL": srv.URL,
		"KUI_HOME":        home,
	}, "hello")

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if stdout != "noconfig answer\n" {
		t.Errorf("stdout = %q, want %q", stdout, "noconfig answer\n")
	}
	// Verify no MCP tools were sent (none configured).
	names := extractToolNames(capturedTools)
	for _, name := range names {
		if name == "test-server_echo" {
			t.Error("MCP tool test-server_echo found in provider request, but no mcp.yaml was configured")
		}
	}
}

// TestCLIMCPServerDiscoverTools covers REQ-TOOLS-4: when mcp.yaml points to
// a valid MCP server, the CLI discovers its tools and sends them to the
// provider alongside built-in tools.
func TestCLIMCPServerDiscoverTools(t *testing.T) {
	var capturedTools []map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Tools []map[string]interface{} `json:"tools"`
		}
		if err := json.Unmarshal(body, &req); err == nil {
			capturedTools = req.Tools
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"mcp tools answer"}}]}`)
	}))
	defer srv.Close()

	fakeServer := compileFakeMCPServer(t)

	home := t.TempDir()
	writeMCPConfig(t, home, "test-server", []string{fakeServer})

	stdout, stderr, code := runCLI(t, map[string]string{
		"OPENAI_API_KEY":  "sk-test-123",
		"OPENAI_BASE_URL": srv.URL,
		"KUI_HOME":        home,
	}, "hello")

	t.Logf("stderr: %q", stderr)
	t.Logf("captured tools count: %d", len(capturedTools))

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if stdout != "mcp tools answer\n" {
		t.Errorf("stdout = %q, want %q", stdout, "mcp tools answer\n")
	}

	// Verify MCP tool was discovered and sent to the provider.
	names := extractToolNames(capturedTools)
	found := false
	for _, name := range names {
		if name == "test-server_echo" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("MCP tool test-server_echo not found in provider request; tool names: %v", names)
	}
}

// TestCLIMCPFailureNonFatal covers the requirement that MCP connection failures
// are non-fatal: if the MCP server cannot be started, the CLI still runs with
// built-in tools only and completes the prompt.
func TestCLIMCPFailureNonFatal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"mcp failover answer"}}]}`)
	}))
	defer srv.Close()

	home := t.TempDir()
	writeMCPConfig(t, home, "broken-server", []string{"nonexistent-binary-that-will-fail"})

	stdout, stderr, code := runCLI(t, map[string]string{
		"OPENAI_API_KEY":  "sk-test-123",
		"OPENAI_BASE_URL": srv.URL,
		"KUI_HOME":        home,
	}, "hello")

	if code != 0 {
		t.Errorf("exit code = %d, want 0 (MCP failure is non-fatal); stderr = %q", code, stderr)
	}
	if stdout != "mcp failover answer\n" {
		t.Errorf("stdout = %q, want the answer despite MCP failure", stdout)
	}
}

// TestCLIMCPToolSchemaValidates covers the requirement that MCP tool schemas
// are valid JSON and the CLI doesn't crash when encountering MCP tool schemas.
func TestCLIMCPToolSchemaValidates(t *testing.T) {
	var capturedRequest string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		capturedRequest = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"schema ok"}}]}`)
	}))
	defer srv.Close()

	fakeServer := compileFakeMCPServer(t)

	home := t.TempDir()
	writeMCPConfig(t, home, "test-server", []string{fakeServer})

	_, stderr, code := runCLI(t, map[string]string{
		"OPENAI_API_KEY":  "sk-test-123",
		"OPENAI_BASE_URL": srv.URL,
		"KUI_HOME":        home,
	}, "hello")

	t.Logf("stderr: %q", stderr)

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr = %q", code, stderr)
	}

	// The captured request must be valid JSON with a tools array containing
	// the MCP tool with a valid JSON schema.
	var req struct {
		Tools []struct {
			Type     string `json:"type"`
			Function struct {
				Name       string          `json:"name"`
				Parameters json.RawMessage `json:"parameters"`
			} `json:"function"`
		} `json:"tools"`
	}
	if err := json.Unmarshal([]byte(capturedRequest), &req); err != nil {
		t.Fatalf("captured request is not valid JSON: %v", err)
	}

	var foundSchema bool
	for _, tool := range req.Tools {
		if tool.Function.Name == "test-server_echo" {
			// Verify the schema is valid JSON.
			var schema map[string]interface{}
			if err := json.Unmarshal(tool.Function.Parameters, &schema); err != nil {
				t.Errorf("MCP tool schema is not valid JSON: %v (raw: %s)", err, tool.Function.Parameters)
			}
			foundSchema = true
		}
	}
	if !foundSchema {
		t.Errorf("MCP tool test-server_echo not found in provider request tools")
	}
}
