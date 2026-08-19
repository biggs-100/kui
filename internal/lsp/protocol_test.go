package lsp

import (
	"encoding/json"
	"testing"
)

func TestPositionJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		pos  Position
	}{
		{"zero position", Position{Line: 0, Character: 0}},
		{"non-zero position", Position{Line: 10, Character: 25}},
		{"large values", Position{Line: 9999, Character: 888}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.pos)
			if err != nil {
				t.Fatalf("marshal error: %v", err)
			}

			var got Position
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}

			if got != tt.pos {
				t.Errorf("round-trip mismatch: got %+v, want %+v", got, tt.pos)
			}
		})
	}
}

func TestDiagnosticJSONRoundTrip(t *testing.T) {
	diag := Diagnostic{
		Range: Range{
			Start: Position{Line: 1, Character: 0},
			End:   Position{Line: 1, Character: 10},
		},
		Severity: DiagnosticSeverityError,
		Message:  "undefined: foo",
		Source:   "gopls",
		Code:     "undefinedVar",
	}

	data, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var got Diagnostic
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if got.Range != diag.Range {
		t.Errorf("range: got %+v, want %+v", got.Range, diag.Range)
	}
	if got.Severity != diag.Severity {
		t.Errorf("severity: got %d, want %d", got.Severity, diag.Severity)
	}
	if got.Message != diag.Message {
		t.Errorf("message: got %q, want %q", got.Message, diag.Message)
	}
	if got.Source != diag.Source {
		t.Errorf("source: got %q, want %q", got.Source, diag.Source)
	}
	if got.Code != diag.Code {
		t.Errorf("code: got %q, want %q", got.Code, diag.Code)
	}
}

func TestHoverJSONRoundTrip(t *testing.T) {
	h := Hover{
		Contents: "func foo() string",
		Range: &Range{
			Start: Position{Line: 5, Character: 2},
			End:   Position{Line: 5, Character: 5},
		},
	}

	data, err := json.Marshal(h)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var got Hover
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if got.Contents != h.Contents {
		t.Errorf("contents: got %q, want %q", got.Contents, h.Contents)
	}
	if got.Range == nil {
		t.Fatal("range is nil, want non-nil")
	}
	if got.Range.Start != h.Range.Start {
		t.Errorf("range start: got %+v, want %+v", got.Range.Start, h.Range.Start)
	}
}

func TestLocationJSONRoundTrip(t *testing.T) {
	loc := Location{
		URI: "file:///tmp/test.go",
		Range: Range{
			Start: Position{Line: 0, Character: 0},
			End:   Position{Line: 0, Character: 5},
		},
	}

	data, err := json.Marshal(loc)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var got Location
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if got.URI != loc.URI {
		t.Errorf("URI: got %q, want %q", got.URI, loc.URI)
	}
}

func TestDiagnosticSeverityConstants(t *testing.T) {
	if DiagnosticSeverityError != 1 {
		t.Errorf("Error = %d, want 1", DiagnosticSeverityError)
	}
	if DiagnosticSeverityWarning != 2 {
		t.Errorf("Warning = %d, want 2", DiagnosticSeverityWarning)
	}
	if DiagnosticSeverityInfo != 3 {
		t.Errorf("Info = %d, want 3", DiagnosticSeverityInfo)
	}
	if DiagnosticSeverityHint != 4 {
		t.Errorf("Hint = %d, want 4", DiagnosticSeverityHint)
	}
}

func TestReferenceParamsJSONRoundTrip(t *testing.T) {
	params := ReferenceParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///tmp/test.go"},
		Position:     Position{Line: 10, Character: 5},
		Context:      ReferenceContext{IncludeDeclaration: true},
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var got ReferenceParams
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if got.TextDocument.URI != params.TextDocument.URI {
		t.Errorf("URI: got %q, want %q", got.TextDocument.URI, params.TextDocument.URI)
	}
	if got.Context.IncludeDeclaration != true {
		t.Errorf("IncludeDeclaration: got false, want true")
	}
}
