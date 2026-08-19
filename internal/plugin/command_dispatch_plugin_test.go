package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

// mockPluginClient implements a minimal interface for testing plugin subprocess communication.
type mockPluginClient struct {
	callFunc func(ctx context.Context, method string, params interface{}) (json.RawMessage, error)
}

func (m *mockPluginClient) CallTool(ctx context.Context, name string, args json.RawMessage) (string, error) {
	if m.callFunc != nil {
		result, err := m.callFunc(ctx, name, args)
		if err != nil {
			return "", err
		}
		var toolResult struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		}
		if err := json.Unmarshal(result, &toolResult); err != nil {
			return "", fmt.Errorf("unmarshal: %w", err)
		}
		if len(toolResult.Content) > 0 {
			return toolResult.Content[0].Text, nil
		}
		return "", nil
	}
	return "", fmt.Errorf("mock not configured")
}

func TestExecutePluginCommand(t *testing.T) {
	mock := &mockPluginClient{
		callFunc: func(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
			// Verify the method is passed through as-is
			if method != "mycommand" {
				t.Errorf("method = %q, want %q", method, "mycommand")
			}
			return json.Marshal(map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": "plugin result"},
				},
			})
		},
	}

	d := NewPluginDiscovery(t.TempDir())
	r := NewPluginRegistry(d)
	cd := NewCommandDispatcher(r)
	pcd := NewPluginCommandDispatcher(cd, mock)

	plugin := &Plugin{
		Manifest: PluginManifest{
			Name:       "test-plugin",
			Version:    "1.0.0",
			Type:       PluginCommand,
			EntryPoint: "./test",
		},
		State: PluginStateLoaded,
	}

	result, err := pcd.ExecutePluginCommand(plugin, "mycommand", []string{"arg1", "arg2"})
	if err != nil {
		t.Fatalf("ExecutePluginCommand() error = %v", err)
	}
	if result != "plugin result" {
		t.Errorf("ExecutePluginCommand() = %q, want %q", result, "plugin result")
	}
}

func TestExecutePluginCommandNotRunning(t *testing.T) {
	d := NewPluginDiscovery(t.TempDir())
	r := NewPluginRegistry(d)
	cd := NewCommandDispatcher(r)
	pcd := NewPluginCommandDispatcher(cd, nil)

	plugin := &Plugin{
		Manifest: PluginManifest{
			Name:       "stopped-plugin",
			Version:    "1.0.0",
			Type:       PluginCommand,
			EntryPoint: "./test",
		},
		State: PluginStateError, // not running
	}

	_, err := pcd.ExecutePluginCommand(plugin, "cmd", nil)
	if err == nil {
		t.Fatal("expected error for non-running plugin, got nil")
	}
}

func TestExecutePluginCommandError(t *testing.T) {
	mock := &mockPluginClient{
		callFunc: func(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
			return nil, fmt.Errorf("subprocess crashed")
		},
	}

	d := NewPluginDiscovery(t.TempDir())
	r := NewPluginRegistry(d)
	cd := NewCommandDispatcher(r)
	pcd := NewPluginCommandDispatcher(cd, mock)

	plugin := &Plugin{
		Manifest: PluginManifest{
			Name:       "crash-plugin",
			Version:    "1.0.0",
			Type:       PluginCommand,
			EntryPoint: "./test",
		},
		State: PluginStateLoaded,
	}

	_, err := pcd.ExecutePluginCommand(plugin, "cmd", nil)
	if err == nil {
		t.Fatal("expected error from crashed subprocess, got nil")
	}
}

func TestExecutePluginCommandJSONRPCParams(t *testing.T) {
	// Verify that the JSON-RPC call includes correct parameters
	mock := &mockPluginClient{
		callFunc: func(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
			// Marshal params to verify structure
			data, err := json.Marshal(params)
			if err != nil {
				t.Fatalf("failed to marshal params: %v", err)
			}

			var parsed map[string]interface{}
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("failed to unmarshal params: %v", err)
			}

			// Verify args are included
			args, ok := parsed["args"].([]interface{})
			if !ok {
				t.Errorf("params.args not found or not array: %v", parsed)
			} else if len(args) != 2 {
				t.Errorf("params.args length = %d, want 2", len(args))
			}

			// Return success
			return json.Marshal(map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": "ok"},
				},
			})
		},
	}

	d := NewPluginDiscovery(t.TempDir())
	r := NewPluginRegistry(d)
	cd := NewCommandDispatcher(r)
	pcd := NewPluginCommandDispatcher(cd, mock)

	plugin := &Plugin{
		Manifest: PluginManifest{
			Name:       "param-plugin",
			Version:    "1.0.0",
			Type:       PluginCommand,
			EntryPoint: "./test",
		},
		State: PluginStateLoaded,
	}

	result, err := pcd.ExecutePluginCommand(plugin, "test-cmd", []string{"a", "b"})
	if err != nil {
		t.Fatalf("ExecutePluginCommand() error = %v", err)
	}
	if result != "ok" {
		t.Errorf("result = %q, want %q", result, "ok")
	}
}

func TestExecutePluginCommandNilClient(t *testing.T) {
	d := NewPluginDiscovery(t.TempDir())
	r := NewPluginRegistry(d)
	cd := NewCommandDispatcher(r)
	pcd := NewPluginCommandDispatcher(cd, nil)

	plugin := &Plugin{
		Manifest: PluginManifest{
			Name:       "no-client-plugin",
			Version:    "1.0.0",
			Type:       PluginCommand,
			EntryPoint: "./test",
		},
		State: PluginStateLoaded,
	}

	_, err := pcd.ExecutePluginCommand(plugin, "cmd", nil)
	if err == nil {
		t.Fatal("expected error for nil client, got nil")
	}
}
