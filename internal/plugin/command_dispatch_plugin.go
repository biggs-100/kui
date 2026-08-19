package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// PluginSubprocessClient is the interface needed to communicate with a plugin
// subprocess via JSON-RPC 2.0. This is a subset of the dynamic.ClientInterface
// focused on command execution.
type PluginSubprocessClient interface {
	// CallTool sends a JSON-RPC 2.0 call to the plugin subprocess.
	CallTool(ctx context.Context, name string, args json.RawMessage) (string, error)
}

// PluginCommandDispatcher bridges a running plugin subprocess to the
// CommandDispatcher. It sends JSON-RPC 2.0 calls to the subprocess
// and returns the result.
type PluginCommandDispatcher struct {
	dispatcher *CommandDispatcher
	client     PluginSubprocessClient
}

// NewPluginCommandDispatcher creates a new bridge between the command
// dispatcher and plugin subprocesses. If client is nil, ExecutePluginCommand
// will return an error for all calls.
func NewPluginCommandDispatcher(dispatcher *CommandDispatcher, client PluginSubprocessClient) *PluginCommandDispatcher {
	return &PluginCommandDispatcher{
		dispatcher: dispatcher,
		client:     client,
	}
}

// ExecutePluginCommand sends a JSON-RPC 2.0 call to a plugin subprocess
// to execute a command. The method name is prefixed with "extensions/" to
// match the existing protocol convention.
func (pcd *PluginCommandDispatcher) ExecutePluginCommand(plugin *Plugin, method string, args []string) (string, error) {
	if pcd.client == nil {
		return "", fmt.Errorf("plugin %q: no subprocess client available", plugin.Manifest.Name)
	}

	if plugin.State != PluginStateLoaded {
		return "", fmt.Errorf("plugin %q is not running (state: %s)", plugin.Manifest.Name, plugin.State)
	}

	// Build JSON-RPC params
	params := map[string]interface{}{
		"plugin": plugin.Manifest.Name,
		"method": method,
		"args":   args,
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return "", fmt.Errorf("plugin %q: marshal params: %w", plugin.Manifest.Name, err)
	}

	// Send JSON-RPC call via the subprocess client
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := pcd.client.CallTool(ctx, method, paramsJSON)
	if err != nil {
		return "", fmt.Errorf("plugin %q: call %q: %w", plugin.Manifest.Name, method, err)
	}

	return result, nil
}

// RegisterPluginCommand registers a plugin command in the dispatcher
// and optionally sends a notification to the plugin subprocess.
func (pcd *PluginCommandDispatcher) RegisterPluginCommand(plugin *Plugin, name, description, args string) error {
	if plugin == nil {
		return fmt.Errorf("cannot register command for nil plugin")
	}

	// Create a handler that delegates to the subprocess
	handler := func(cmdArgs []string) (string, error) {
		return pcd.ExecutePluginCommand(plugin, name, cmdArgs)
	}

	return pcd.dispatcher.Register(name, plugin.Manifest.Name, description, args, handler)
}

// UnregisterPluginCommands removes all commands belonging to a plugin
// from the dispatcher. This should be called during plugin shutdown.
func (pcd *PluginCommandDispatcher) UnregisterPluginCommands(plugin *Plugin) int {
	if plugin == nil {
		return 0
	}
	return pcd.dispatcher.UnregisterByPlugin(plugin.Manifest.Name)
}
