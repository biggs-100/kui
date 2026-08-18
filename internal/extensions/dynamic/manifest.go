package dynamic

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Manifest describes a single dynamic extension discovered from an
// extension.yaml file. The system scans global and project directories for
// these manifests to build the runtime extension registry.
type Manifest struct {
	Name            string `yaml:"name"`
	Version         string `yaml:"version"`
	ProtocolVersion string `yaml:"protocol_version"`
	EntryPoint      string `yaml:"entry_point"`
}

// LoadManifest reads and parses an extension.yaml file into a Manifest.
// It validates that all required fields are present.
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &ManifestError{
			File: path,
			Err:  fmt.Errorf("read: %w", err),
		}
	}

	if len(data) == 0 {
		return nil, &ManifestError{
			File: path,
			Err:  fmt.Errorf("empty manifest"),
		}
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, &ManifestError{
			File: path,
			Err:  fmt.Errorf("parse: %w", err),
		}
	}

	if err := validateManifest(path, &m); err != nil {
		return nil, err
	}

	return &m, nil
}

// validateManifest checks that all required fields are present.
func validateManifest(path string, m *Manifest) error {
	if m.Name == "" {
		return &ManifestError{File: path, Field: "name", Err: fmt.Errorf("required")}
	}
	if m.Version == "" {
		return &ManifestError{File: path, Field: "version", Err: fmt.Errorf("required")}
	}
	if m.ProtocolVersion == "" {
		return &ManifestError{File: path, Field: "protocol_version", Err: fmt.Errorf("required")}
	}
	if m.EntryPoint == "" {
		return &ManifestError{File: path, Field: "entry_point", Err: fmt.Errorf("required")}
	}
	return nil
}
