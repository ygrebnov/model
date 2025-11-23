package validation

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// helper constructor
func fe(path, rule, msg string) FieldError {
	var err error
	if msg != "" {
		err = errors.New(msg)
	}
	return FieldError{Path: path, Rule: rule, Err: err}
}

func TestValidationError_Add_and_Len_Empty_nilReceiverSafe(t *testing.T) {
	t.Parallel()

	// nil receiver: Add should be a no-op and not panic; Len/Empty should be safe
	var veNil *Error
	veNil.Add(fe("A", "r", "x")) // must not panic
	if veNil.Len() != 0 {
		t.Fatalf("nil receiver Len() = %d, want 0", veNil.Len())
	}
	if !veNil.Empty() {
		t.Fatalf("nil receiver Empty() = false, want true")
	}

	// non-nil receiver
	ve := &Error{}
	if ve.Len() != 0 || !ve.Empty() {
		t.Fatalf("initial Len/Empty wrong: Len=%d Empty=%v", ve.Len(), ve.Empty())
	}
	ve.Add(fe("A", "r1", "x"))
	ve.Add(fe("B", "r2", "y"))
	if ve.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", ve.Len())
	}
	if ve.Empty() {
		t.Fatalf("Empty() = true, want false")
	}
}

func TestValidationError_Addf(t *testing.T) {
	t.Parallel()

	ve := &Error{}
	ve.Addf("Root.Name", "nonempty", errors.New("must not be empty"))
	if ve.Len() != 1 {
		t.Fatalf("Len() = %d, want 1", ve.Len())
	}
	got := ve.ForField("Root.Name")
	if len(got) != 1 || got[0].Rule != "nonempty" || got[0].Err == nil {
		t.Fatalf("Addf did not record expected FieldError: %+v", got)
	}
}

func TestValidationError_ErrorFormatting(t *testing.T) {
	t.Parallel()

	// nil receiver → empty string
	var veNil *Error
	if s := veNil.Error(); s != "" {
		t.Fatalf("nil receiver Error() = %q, want empty", s)
	}

	// 0 issues → empty
	ve0 := &Error{}
	if s := ve0.Error(); s != "" {
		t.Fatalf("0 issues Error() = %q, want empty", s)
	}

	// 1 issue → single line (no header/footer)
	ve1 := &Error{}
	ve1.Add(fe("Name", "nonempty", "must not be empty"))
	s1 := ve1.Error()
	if !strings.Contains(s1, "Name") || !strings.Contains(s1, "must not be empty") {
		t.Fatalf("single issue Error() missing content: %q", s1)
	}
	if strings.Contains(s1, "validation failed (") {
		t.Fatalf("single issue should not contain header/footer: %q", s1)
	}

	// 2+ issues → multi-line with header/footer and each line
	ve2 := &Error{}
	ve2.Add(fe("Name", "nonempty", "x"))
	ve2.Add(fe("Age", "positive", "y"))
	s2 := ve2.Error()
	if !strings.HasPrefix(s2, "validation failed") {
		t.Fatalf("multi Error() missing header/footer: %q", s2)
	}
	if !strings.Contains(s2, "Name") || !strings.Contains(s2, "Age") {
		t.Fatalf("multi Error() missing entries: %q", s2)
	}
}

func TestValidationError_Unwrap(t *testing.T) {
	t.Parallel()

	// nil receiver -> nil
	var veNil *Error
	if err := veNil.Unwrap(); err != nil {
		t.Fatalf("nil receiver Unwrap() = %v, want nil", err)
	}

	// no underlying errs -> Join(nil...) => nil
	ve0 := &Error{}
	if err := ve0.Unwrap(); err != nil {
		t.Fatalf("empty Unwrap() = %v, want nil", err)
	}

	// with underlying errs (including one nil) -> Is works
	e1 := errors.New("e1")
	var eNil error
	ve := &Error{}
	ve.Add(FieldError{Path: "A", Err: e1})
	ve.Add(FieldError{Path: "B", Err: eNil})
	u := ve.Unwrap()
	if !errors.Is(u, e1) {
		t.Fatalf("Unwrap() should contain e1 via errors.Is")
	}
}

func TestValidationError_ForField_and_ByField_and_Fields(t *testing.T) {
	t.Parallel()

	// nil receiver
	var veNil *Error
	if got := veNil.ForField("X"); got != nil {
		t.Fatalf("nil ve ForField returned %v, want nil", got)
	}
	if m := veNil.ByField(); len(m) != 0 {
		t.Fatalf("nil ve ByField non-empty map: %+v", m)
	}
	if f := veNil.Fields(); f != nil {
		t.Fatalf("nil ve Fields returned %v, want nil", f)
	}

	// non-nil: add multiple issues, including duplicates and multiple fields
	ve := &Error{}
	ve.Add(fe("A", "r1", "x"))
	ve.Add(fe("B", "r2", "y1"))
	ve.Add(fe("B", "r3", "y2"))
	ve.Add(fe("A", "r4", "x2"))

	// ForField exact match preserves order of additions for that field
	a := ve.ForField("A")
	if len(a) != 2 || a[0].Rule != "r1" || a[1].Rule != "r4" {
		t.Fatalf("ForField(A) wrong: %+v", a)
	}
	// Missing field -> empty
	if c := ve.ForField("C"); len(c) != 0 {
		t.Fatalf("ForField(C) should be empty, got %+v", c)
	}

	// ByField groups correctly
	m := ve.ByField()
	if len(m) != 2 || len(m["A"]) != 2 || len(m["B"]) != 2 {
		t.Fatalf("ByField grouping wrong: %+v", m)
	}

	// Fields preserves first-seen order and uniqueness
	paths := ve.Fields()
	want := []string{"A", "B"}
	if len(paths) != len(want) || paths[0] != want[0] || paths[1] != want[1] {
		t.Fatalf("Fields order/contents wrong: got %v want %v", paths, want)
	}
}

func TestValidationError_MarshalJSON(t *testing.T) {
	t.Parallel()

	// nil receiver -> "null"
	var veNil *Error
	b, err := veNil.MarshalJSON()
	if err != nil {
		t.Fatalf("nil ve MarshalJSON error: %v", err)
	}
	if string(b) != "null" {
		t.Fatalf("nil ve MarshalJSON = %s, want null", string(b))
	}

	// non-nil with multiple fields and nil Err message handling
	ve := &Error{}
	ve.Add(fe("A", "r1", "x"))
	ve.Add(fe("B", "r2", "")) // nil underlying Err → empty message
	ve.Add(fe("B", "r3", "y"))

	data, err := ve.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON error: %v", err)
	}
	var m map[string][]string
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v; raw=%s", err, string(data))
	}
	if len(m) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(m))
	}
	if got := m["A"]; len(got) != 1 || got[0] != "x" {
		t.Fatalf("A wrong: %#v", got)
	}
	if got := m["B"]; len(got) != 2 || got[0] != "" || got[1] != "y" {
		t.Fatalf("B wrong: %#v", got)
	}
}

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
			wantHas: []string{"Field \"Root.Name\"", "rule \"nonempty\"", "must not be empty"},
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
			wantHas: []string{"Field \"X\"", "rule \"some\""},
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
			name: "full fields with Params and message",
			fe: FieldError{
				Path:   "User.Email",
				Rule:   "nonempty",
				Params: []string{"p1", "p2"},
				Err:    errors.New("must not be empty"),
			},
			wantHas:    []string{`"path":"User.Email"`, `"rule":"nonempty"`, `"Params":["p1","p2"]`, `"message":"must not be empty"`},
			wantNotHas: []string{}, // all present
		},
		{
			name: "no Params should omit Params field",
			fe: FieldError{
				Path: "A",
				Rule: "r",
				Err:  errors.New("x"),
			},
			wantHas:    []string{`"path":"A"`, `"rule":"r"`, `"message":"x"`},
			wantNotHas: []string{`"Params"`},
		},
		{
			name: "nil error produces empty message string",
			fe: FieldError{
				Path:   "B",
				Rule:   "r2",
				Params: []string{"k"},
				Err:    nil,
			},
			wantHas:    []string{`"path":"B"`, `"rule":"r2"`, `"Params":["k"]`, `"message":""`},
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
