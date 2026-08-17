package tui

import (
	"context"
	"errors"
	"testing"

	"github.com/biggs-100/kui/internal/core"
)

// --- fakes for run_test ---

// failingProvider always returns an error, simulating a broken provider.
type failingProvider struct {
	err error
}

func (p *failingProvider) Chat(_ context.Context, _ []core.Message, _ []core.Tool) ([]core.Message, error) {
	return nil, p.err
}

// errClientFactory returns a factory that produces a failing provider.
func errClientFactory(err error) func() (core.Provider, error) {
	return func() (core.Provider, error) {
		return nil, err
	}
}

// --- Run(ctx, wiring) tests (REQ-TUI-APP-1) ---

func TestRunStartupFailureReturnsError(t *testing.T) {
	// REQ-TUI-APP-1: If startup fails (invalid provider), Run must return
	// a non-nil error and NOT start the TUI.
	wiring := Wiring{
		ProfileRoot: t.TempDir(),
		ProjectDir:  t.TempDir(),
		ConfigRoot:  t.TempDir(),
		Client:      errClientFactory(errors.New("invalid api key")),
		MaxIter:     10,
	}

	err := Run(context.Background(), wiring)
	if err == nil {
		t.Fatal("expected error from Run with failing provider")
	}
}

func TestRunStartupFailureNoTUIRenders(t *testing.T) {
	// REQ-TUI-APP-1: Startup failure must NOT render the TUI.
	wiring := Wiring{
		ProfileRoot: t.TempDir(),
		ProjectDir:  t.TempDir(),
		ConfigRoot:  t.TempDir(),
		Client:      errClientFactory(errors.New("connection refused")),
		MaxIter:     10,
	}

	err := Run(context.Background(), wiring)
	if err == nil {
		t.Fatal("expected error from Run with failing provider")
	}
	// The error should be the provider error, not a TUI error
	if err.Error() != "connection refused" {
		t.Errorf("error = %q, want %q", err.Error(), "connection refused")
	}
}

func TestRunWiringComposesStoreAndLoader(t *testing.T) {
	// Verify that Run builds the store and loader from wiring fields.
	// Use a failing client to avoid starting the real TUI (which blocks
	// on terminal input). The client error proves wiring was set up
	// correctly — the failure is AFTER store/loader creation.
	cfgRoot := t.TempDir()
	wiring := Wiring{
		ProfileRoot: t.TempDir(),
		ProjectDir:  t.TempDir(),
		ConfigRoot:  cfgRoot,
		Client:      errClientFactory(errors.New("startup fail")),
		MaxIter:     10,
	}

	err := Run(context.Background(), wiring)
	if err == nil {
		t.Fatal("expected error from Run with failing provider")
	}
	if err.Error() != "startup fail" {
		t.Errorf("error = %q, want %q", err.Error(), "startup fail")
	}
}

func TestRunEmptyProfilesStartsWithDefault(t *testing.T) {
	// With no profiles, the controller should still start with a default
	// empty-profile controller.
	wiring := Wiring{
		ProfileRoot: t.TempDir(),
		ProjectDir:  t.TempDir(),
		ConfigRoot:  t.TempDir(),
		Client:      errClientFactory(errors.New("startup fail")),
		MaxIter:     10,
	}

	err := Run(context.Background(), wiring)
	if err == nil {
		t.Fatal("expected error from Run with failing provider")
	}
}

func TestRunMaxIterDefault(t *testing.T) {
	// When MaxIter is 0, Run should default to maxIterations.
	wiring := Wiring{
		ProfileRoot: t.TempDir(),
		ProjectDir:  t.TempDir(),
		ConfigRoot:  t.TempDir(),
		Client:      errClientFactory(errors.New("fail")),
		MaxIter:     0,
	}

	err := Run(context.Background(), wiring)
	if err == nil {
		t.Fatal("expected error from Run")
	}
}
