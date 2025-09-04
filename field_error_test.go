package model

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestFieldError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		fe      FieldError
		wantHas []string // substrings that must be present
		wantNot []string // substrings that must be absent
	}{
		{
			name: "with rule and non-nil error",
			fe: FieldError{
				Path: "Root.Name",
				Rule: "nonempty",
				Err:  errors.New("must not be empty"),
			},
			wantHas: []string{"Root.Name", "must not be empty", "(rule nonempty)"},
		},
		{
			name: "without rule and non-nil error",
			fe: FieldError{
				Path: "Wait",
				Err:  errors.New("non-zero required"),
			},
			wantHas: []string{"Wait", "non-zero required"},
			wantNot: []string{"(rule"},
		},
		{
			name: "with rule and nil error (should still include path and rule, no panic)",
			fe: FieldError{
				Path: "X",
				Rule: "some",
				Err:  nil,
			},
			// Implementation currently formats the nil error; we only assert it contains path and rule marker.
			wantHas: []string{"X", "(rule some)"},
		},
		{
			name: "without rule and nil error (should still include path, no panic)",
			fe: FieldError{
				Path: "Field",
				Err:  nil,
			},
			wantHas: []string{"Field"},
			wantNot: []string{"(rule"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fe.Error()
			for _, s := range tt.wantHas {
				if !strings.Contains(got, s) {
					t.Fatalf("Error() missing %q in %q", s, got)
				}
			}
			for _, s := range tt.wantNot {
				if strings.Contains(got, s) {
					t.Fatalf("Error() unexpectedly contains %q in %q", s, got)
				}
			}
		})
	}
}

func TestFieldError_Unwrap(t *testing.T) {
	t.Parallel()

	e := errors.New("boom")
	feWith := FieldError{Path: "P", Rule: "r", Err: e}
	feNil := FieldError{Path: "Q", Rule: "r2", Err: nil}

	if un := feWith.Unwrap(); !errors.Is(un, e) {
		t.Fatalf("Unwrap() = %v, want original error", un)
	}
	if un := feNil.Unwrap(); un != nil {
		t.Fatalf("Unwrap() = %v, want nil", un)
	}
}

func TestFieldError_MarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		fe         FieldError
		wantHas    []string // substrings expected in JSON
		wantNotHas []string // substrings that must not appear
	}{
		{
			name: "full fields with params and message",
			fe: FieldError{
				Path:   "User.Email",
				Rule:   "nonempty",
				Params: []string{"p1", "p2"},
				Err:    errors.New("must not be empty"),
			},
			wantHas:    []string{`"path":"User.Email"`, `"rule":"nonempty"`, `"params":["p1","p2"]`, `"message":"must not be empty"`},
			wantNotHas: []string{}, // all present
		},
		{
			name: "no params should omit params field",
			fe: FieldError{
				Path: "A",
				Rule: "r",
				Err:  errors.New("x"),
			},
			wantHas:    []string{`"path":"A"`, `"rule":"r"`, `"message":"x"`},
			wantNotHas: []string{`"params"`},
		},
		{
			name: "nil error produces empty message string",
			fe: FieldError{
				Path:   "B",
				Rule:   "r2",
				Params: []string{"k"},
				Err:    nil,
			},
			wantHas:    []string{`"path":"B"`, `"rule":"r2"`, `"params":["k"]`, `"message":""`},
			wantNotHas: []string{},
		},
		{
			name: "empty rule allowed",
			fe: FieldError{
				Path: "C",
				Rule: "",
				Err:  errors.New("oops"),
			},
			wantHas:    []string{`"path":"C"`, `"rule":""`, `"message":"oops"`},
			wantNotHas: []string{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.fe.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON error: %v", err)
			}
			js := string(data)
			// ensure valid JSON
			var tmp map[string]any
			if err := json.Unmarshal(data, &tmp); err != nil {
				t.Fatalf("invalid JSON: %v, raw: %s", err, js)
			}
			for _, s := range tt.wantHas {
				if !strings.Contains(js, s) {
					t.Fatalf("JSON missing %q in %s", s, js)
				}
			}
			for _, s := range tt.wantNotHas {
				if strings.Contains(js, s) {
					t.Fatalf("JSON unexpectedly contains %q in %s", s, js)
				}
			}
		})
	}
}
