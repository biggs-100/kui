package subagent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePolicyValid(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want Policy
	}{
		{"on", `{"schema":"kui.background-subagents/v1","policy":"on"}`, PolicyOn},
		{"off", `{"schema":"kui.background-subagents/v1","policy":"off"}`, PolicyOff},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePolicy(tt.raw)
			if err != nil {
				t.Fatalf("ParsePolicy() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("ParsePolicy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParsePolicyInvalid(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{"bad json", `{malformed`},
		{"wrong schema", `{"schema":"wrong","policy":"on"}`},
		{"bad policy", `{"schema":"kui.background-subagents/v1","policy":"invalid"}`},
		{"extra keys", `{"schema":"kui.background-subagents/v1","policy":"on","extra":1}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParsePolicy(tt.raw)
			if err == nil {
				t.Error("ParsePolicy() expected error, got nil")
			}
		})
	}
}

func TestResolvePolicyProjectFile(t *testing.T) {
	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "project")
	globalDir := filepath.Join(tmp, "global")
	os.MkdirAll(projectDir, 0755)

	// Write project file.
	os.WriteFile(filepath.Join(projectDir, "background-subagents.json"),
		[]byte(`{"schema":"kui.background-subagents/v1","policy":"on"}`), 0644)

	policy, source := ResolvePolicy(projectDir, globalDir)
	if policy != PolicyOn {
		t.Errorf("policy = %v, want on", policy)
	}
	if source != "project_file" {
		t.Errorf("source = %v, want project_file", source)
	}
}

func TestResolvePolicyGlobalFile(t *testing.T) {
	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "project")
	globalDir := filepath.Join(tmp, "global")
	os.MkdirAll(globalDir, 0755)

	// Write global file only.
	os.WriteFile(filepath.Join(globalDir, "background-subagents.json"),
		[]byte(`{"schema":"kui.background-subagents/v1","policy":"on"}`), 0644)

	policy, source := ResolvePolicy(projectDir, globalDir)
	if policy != PolicyOn {
		t.Errorf("policy = %v, want on", policy)
	}
	if source != "global_file" {
		t.Errorf("source = %v, want global_file", source)
	}
}

func TestResolvePolicyDefault(t *testing.T) {
	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "project")
	globalDir := filepath.Join(tmp, "global")

	policy, source := ResolvePolicy(projectDir, globalDir)
	if policy != PolicyOff {
		t.Errorf("policy = %v, want off", policy)
	}
	if source != "default" {
		t.Errorf("source = %v, want default", source)
	}
}

func TestResolvePolicyMalformedProjectFile(t *testing.T) {
	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "project")
	globalDir := filepath.Join(tmp, "global")
	os.MkdirAll(projectDir, 0755)

	// Write malformed project file.
	os.WriteFile(filepath.Join(projectDir, "background-subagents.json"),
		[]byte(`{malformed`), 0644)

	policy, source := ResolvePolicy(projectDir, globalDir)
	if policy != PolicyOff {
		t.Errorf("policy = %v, want off (fail closed)", policy)
	}
	if source != "project_file_malformed" {
		t.Errorf("source = %v, want project_file_malformed", source)
	}
}

func TestResolveCapability(t *testing.T) {
	tests := []struct {
		name  string
		tools []string
		want  Capability
	}{
		{"has subagent_run", []string{"read", "bash", "subagent_run"}, CapabilityReady},
		{"has namespaced", []string{"read", "pi-subagents.subagent_run"}, CapabilityReady},
		{"no subagent_run", []string{"read", "bash"}, CapabilityAbsent},
		{"empty tools", []string{}, CapabilityAbsent},
		{"nil tools", nil, CapabilityAbsent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveCapability(tt.tools)
			if got != tt.want {
				t.Errorf("ResolveCapability() = %v, want %v", got, tt.want)
			}
		})
	}
}
