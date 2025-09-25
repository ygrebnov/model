package model

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

	// nil receiver: add should be a no-op and not panic; Len/Empty should be safe
	var veNil *ValidationError
	veNil.Add(fe("A", "r", "x")) // must not panic
	if veNil.Len() != 0 {
		t.Fatalf("nil receiver Len() = %d, want 0", veNil.Len())
	}
	if !veNil.Empty() {
		t.Fatalf("nil receiver Empty() = false, want true")
	}

	// non-nil receiver
	ve := &ValidationError{}
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

	ve := &ValidationError{}
	ve.Addf("Root.name", "nonempty", errors.New("must not be empty"))
	if ve.Len() != 1 {
		t.Fatalf("Len() = %d, want 1", ve.Len())
	}
	got := ve.ForField("Root.name")
	if len(got) != 1 || got[0].Rule != "nonempty" || got[0].Err == nil {
		t.Fatalf("Addf did not record expected FieldError: %+v", got)
	}
}

func TestValidationError_ErrorFormatting(t *testing.T) {
	t.Parallel()

	// nil receiver → empty string
	var veNil *ValidationError
	if s := veNil.Error(); s != "" {
		t.Fatalf("nil receiver Error() = %q, want empty", s)
	}

	// 0 issues → empty
	ve0 := &ValidationError{}
	if s := ve0.Error(); s != "" {
		t.Fatalf("0 issues Error() = %q, want empty", s)
	}

	// 1 issue → single line (no header/footer)
	ve1 := &ValidationError{}
	ve1.Add(fe("name", "nonempty", "must not be empty"))
	s1 := ve1.Error()
	if !strings.Contains(s1, "name") || !strings.Contains(s1, "must not be empty") {
		t.Fatalf("single issue Error() missing content: %q", s1)
	}
	if strings.Contains(s1, "validation failed (") {
		t.Fatalf("single issue should not contain header/footer: %q", s1)
	}

	// 2+ issues → multi-line with header/footer and each line
	ve2 := &ValidationError{}
	ve2.Add(fe("name", "nonempty", "x"))
	ve2.Add(fe("Age", "positive", "y"))
	s2 := ve2.Error()
	if !strings.HasPrefix(s2, "validation failed (") || !strings.HasSuffix(s2, "\n)") {
		t.Fatalf("multi Error() missing header/footer: %q", s2)
	}
	if !strings.Contains(s2, "name:") || !strings.Contains(s2, "Age:") {
		t.Fatalf("multi Error() missing entries: %q", s2)
	}
}

func TestValidationError_Unwrap(t *testing.T) {
	t.Parallel()

	// nil receiver -> nil
	var veNil *ValidationError
	if err := veNil.Unwrap(); err != nil {
		t.Fatalf("nil receiver Unwrap() = %v, want nil", err)
	}

	// no underlying errs -> Join(nil...) => nil
	ve0 := &ValidationError{}
	if err := ve0.Unwrap(); err != nil {
		t.Fatalf("empty Unwrap() = %v, want nil", err)
	}

	// with underlying errs (including one nil) -> Is works
	e1 := errors.New("e1")
	var eNil error
	ve := &ValidationError{}
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
	var veNil *ValidationError
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
	ve := &ValidationError{}
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
	var veNil *ValidationError
	b, err := veNil.MarshalJSON()
	if err != nil {
		t.Fatalf("nil ve MarshalJSON error: %v", err)
	}
	if string(b) != "null" {
		t.Fatalf("nil ve MarshalJSON = %s, want null", string(b))
	}

	// non-nil with multiple fields and nil Err message handling
	ve := &ValidationError{}
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
