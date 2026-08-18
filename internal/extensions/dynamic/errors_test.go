package dynamic

import (
	"errors"
	"testing"
)

func TestManifestError(t *testing.T) {
	err := &ManifestError{File: "ext.yaml", Field: "name", Err: errors.New("required")}
	if err.Error() != `manifest "ext.yaml": field "name": required` {
		t.Errorf("Error() = %q", err.Error())
	}
	var me *ManifestError
	if !errors.As(err, &me) {
		t.Error("errors.As failed for ManifestError")
	}
	if me.Err.Error() != "required" {
		t.Errorf("Unwrap = %v", me.Err)
	}
}

func TestManifestErrorNoField(t *testing.T) {
	err := &ManifestError{File: "ext.yaml", Err: errors.New("malformed")}
	if err.Error() != `manifest "ext.yaml": malformed` {
		t.Errorf("Error() = %q", err.Error())
	}
}

func TestProtocolError(t *testing.T) {
	err := &ProtocolError{Extension: "notes", Method: "initialize", Err: errors.New("version mismatch")}
	if err.Error() != `extension "notes": protocol error on "initialize": version mismatch` {
		t.Errorf("Error() = %q", err.Error())
	}
	var pe *ProtocolError
	if !errors.As(err, &pe) {
		t.Error("errors.As failed for ProtocolError")
	}
	if pe.Err.Error() != "version mismatch" {
		t.Errorf("Unwrap = %v", pe.Err)
	}
}

func TestSpawnError(t *testing.T) {
	err := &SpawnError{Extension: "notes", EntryPoint: "/usr/bin/notes-ext", Err: errors.New("not found")}
	if err.Error() != `extension "notes": spawn "/usr/bin/notes-ext" failed: not found` {
		t.Errorf("Error() = %q", err.Error())
	}
	var se *SpawnError
	if !errors.As(err, &se) {
		t.Error("errors.As failed for SpawnError")
	}
	if se.Err.Error() != "not found" {
		t.Errorf("Unwrap = %v", se.Err)
	}
}
