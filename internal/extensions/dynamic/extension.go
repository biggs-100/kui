package dynamic

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/biggs-100/kui/internal/core"
)

// ClientFactory is a function that spawns a client subprocess for an extension.
// Extracted so the manager and tests can inject different client constructors.
type ClientFactory func(ctx context.Context, entryPoint string) (ClientInterface, error)

// defaultClientFactory wraps NewClient to satisfy ClientFactory.
func defaultClientFactory(ctx context.Context, entryPoint string) (ClientInterface, error) {
	return NewClient(ctx, entryPoint)
}

// DynamicExtension implements core.Extension by spawning an external
// subprocess and bridging its tools into the kui registry (D1).
type DynamicExtension struct {
	name       string
	entryPoint string
	factory    ClientFactory
	client     ClientInterface
	cancel     context.CancelFunc
}

// Verify DynamicExtension implements core.Extension at compile time.
var _ core.Extension = (*DynamicExtension)(nil)

// NewDynamicExtension creates a DynamicExtension from a manifest. If factory
// is nil, the default real client factory is used.
func NewDynamicExtension(manifest *Manifest, factory ClientFactory) *DynamicExtension {
	if factory == nil {
		factory = defaultClientFactory
	}
	return &DynamicExtension{
		name:       manifest.Name,
		entryPoint: manifest.EntryPoint,
		factory:    factory,
	}
}

// Name returns the extension's stable identifier (D1).
func (e *DynamicExtension) Name() string {
	return e.name
}

// Init spawns the extension subprocess, performs the handshake, discovers
// tools, and registers them via the extension API (REQ-EXT-2).
func (e *DynamicExtension) Init(api core.ExtensionAPI) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	e.cancel = cancel

	// Spawn subprocess.
	client, err := e.factory(ctx, e.entryPoint)
	if err != nil {
		cancel()
		return &SpawnError{
			Extension:  e.name,
			EntryPoint: e.entryPoint,
			Err:        err,
		}
	}
	e.client = client

	// Initialize handshake.
	if err := client.Initialize(ctx); err != nil {
		cancel()
		return &ProtocolError{
			Extension: e.name,
			Method:    "initialize",
			Err:       err,
		}
	}

	// Discover tools.
	tools, err := client.ListTools(ctx)
	if err != nil {
		cancel()
		return fmt.Errorf("extension %q: list tools: %w", e.name, err)
	}

	// Register each tool with the prefix {extensionName}_{toolName}.
	for _, def := range tools {
		tool := NewDynamicTool(e.name, def, client)
		if err := api.RegisterTool(tool); err != nil {
			cancel()
			return fmt.Errorf("extension %q: register tool %q: %w", e.name, def.Name, err)
		}
	}

	log.Printf("extension %q: registered %d tools", e.name, len(tools))
	return nil
}

// Shutdown sends a shutdown notification, waits up to 5 seconds for the
// process to exit, then kills it (REQ-EXT-4).
func (e *DynamicExtension) Shutdown() error {
	if e.cancel != nil {
		defer e.cancel()
	}

	// No client means Init was never called or already shut down.
	if e.client == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return e.client.Shutdown(ctx)
}
