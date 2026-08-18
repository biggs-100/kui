// Package extensions provides compiled-in extension discovery via Go's init()
// self-registration pattern (D6, REQ-DISCOVERY-1). Extensions register at
// import time via Register(), then LoadAll initializes them in registration
// order, and ShutdownAll tears them down in reverse order.
package extensions

import (
	"errors"

	"github.com/biggs-100/kui/internal/core"
)

// global holds extensions registered via init() functions (REQ-DISCOVERY-1).
// The order reflects the Go import graph — deterministic per binary build.
var global []core.Extension

// dynamic holds extensions registered at runtime from discovered extension
// manifests (Phase 4). They are initialized after all global extensions.
var dynamic []core.Extension

// loaded holds extensions that have been successfully initialized by LoadAll.
// ShutdownAll processes this slice in reverse order (REQ-DISCOVERY-4).
var loaded []core.Extension

// Register appends ext to the global registry (REQ-DISCOVERY-2). Calling
// Register with a nil extension panics to fail fast at startup.
func Register(ext core.Extension) {
	if ext == nil {
		panic("extensions: Register called with nil extension")
	}
	global = append(global, ext)
}

// RegisterDynamic appends ext to the dynamic registry. Dynamic extensions are
// initialized after all global extensions during LoadAll. Calling
// RegisterDynamic with a nil extension panics to fail fast.
func RegisterDynamic(ext core.Extension) {
	if ext == nil {
		panic("extensions: RegisterDynamic called with nil extension")
	}
	dynamic = append(dynamic, ext)
}

// LoadAll initializes every registered extension in registration order —
// global extensions first, then dynamic extensions — by calling Init(api) on
// each (REQ-DISCOVERY-3). If any Init returns an error, LoadAll stops, calls
// Shutdown on all previously-initialized extensions in reverse order (rollback),
// and returns the error.
func LoadAll(api core.ExtensionAPI) error {
	all := make([]core.Extension, 0, len(global)+len(dynamic))
	all = append(all, global...)
	all = append(all, dynamic...)

	for i, ext := range all {
		if err := ext.Init(api); err != nil {
			// Rollback: shutdown previously-initialized extensions reverse-order.
			for j := i - 1; j >= 0; j-- {
				_ = all[j].Shutdown()
			}
			return err
		}
	}
	loaded = make([]core.Extension, len(all))
	copy(loaded, all)
	return nil
}

// ShutdownAll calls Shutdown on every loaded extension in reverse registration
// order (REQ-DISCOVERY-4). It is idempotent — calling it twice is a no-op on
// the second call. Errors during Shutdown are collected and returned via
// errors.Join rather than short-circuited.
func ShutdownAll() error {
	var errs []error
	for i := len(loaded) - 1; i >= 0; i-- {
		if err := loaded[i].Shutdown(); err != nil {
			errs = append(errs, err)
		}
	}
	loaded = nil
	return errors.Join(errs...)
}
