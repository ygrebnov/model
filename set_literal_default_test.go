package model

import (
	"reflect"
	"testing"
	"time"
)

type holder struct {
	S   string
	B   bool
	I   int
	U   uint
	F64 float64
	D   time.Duration

	PS *string
	PI *int
	PD *time.Duration

	SL  []int
	MP  map[string]int
	St  struct{ X int }
	PSt *struct{ X int }
}

// field returns a settable reflect.Value for the named field of h.
func field(h *holder, name string) reflect.Value {
	v := reflect.ValueOf(h).Elem().FieldByName(name)
	if !v.IsValid() {
		panic("bad field: " + name)
	}
	return v
}

func TestSetLiteralDefault(t *testing.T) {
	t.Run("string: set when zero", func(t *testing.T) {
		h := &holder{}
		if err := setLiteralDefault(field(h, "S"), "hello"); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if h.S != "hello" {
			t.Fatalf("got %q want %q", h.S, "hello")
		}
	})

	t.Run("string: skip when non-zero", func(t *testing.T) {
		h := &holder{S: "keep"}
		if err := setLiteralDefault(field(h, "S"), "new"); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if h.S != "keep" {
			t.Fatalf("overwrote non-zero: got %q", h.S)
		}
	})

	t.Run("bool: true literals", func(t *testing.T) {
		trueLits := []string{"1", "true", "t", "yes", "y", "on"}
		for _, lit := range trueLits {
			h := &holder{}
			if err := setLiteralDefault(field(h, "B"), lit); err != nil {
				t.Fatalf("%s: %v", lit, err)
			}
			if h.B != true {
				t.Fatalf("%s: got %v want true", lit, h.B)
			}
		}
	})

	t.Run("bool: false literals", func(t *testing.T) {
		falseLits := []string{"0", "false", "f", "no", "n", "off"}
		for _, lit := range falseLits {
			h := &holder{}
			if err := setLiteralDefault(field(h, "B"), lit); err != nil {
				t.Fatalf("%s: %v", lit, err)
			}
			if h.B != false {
				t.Fatalf("%s: got %v want false", lit, h.B)
			}
		}
	})

	t.Run("bool: invalid literal -> error", func(t *testing.T) {
		h := &holder{}
		if err := setLiteralDefault(field(h, "B"), "maybe"); err == nil {
			t.Fatalf("expected error for invalid bool")
		}
	})

	t.Run("int: set & parse error", func(t *testing.T) {
		// set
		{
			h := &holder{}
			if err := setLiteralDefault(field(h, "I"), "42"); err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if h.I != 42 {
				t.Fatalf("got %d want 42", h.I)
			}
		}
		// parse error
		{
			h := &holder{}
			if err := setLiteralDefault(field(h, "I"), "NaN"); err == nil {
				t.Fatalf("expected parse error")
			}
		}
	})

	t.Run("uint: set & parse error (negative)", func(t *testing.T) {
		// set
		{
			h := &holder{}
			if err := setLiteralDefault(field(h, "U"), "7"); err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if h.U != 7 {
				t.Fatalf("got %d want 7", h.U)
			}
		}
		// negative -> parse error
		{
			h := &holder{}
			if err := setLiteralDefault(field(h, "U"), "-1"); err == nil {
				t.Fatalf("expected parse error for negative uint")
			}
		}
	})

	t.Run("float64: set & parse error", func(t *testing.T) {
		// set
		{
			h := &holder{}
			if err := setLiteralDefault(field(h, "F64"), "3.14"); err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if h.F64 != 3.14 {
				t.Fatalf("got %v want 3.14", h.F64)
			}
		}
		// parse error
		{
			h := &holder{}
			if err := setLiteralDefault(field(h, "F64"), "nope"); err == nil {
				t.Fatalf("expected parse error")
			}
		}
	})

	t.Run("duration: set via literal", func(t *testing.T) {
		h := &holder{}
		if err := setLiteralDefault(field(h, "D"), "1h30m"); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if h.D != time.Hour+30*time.Minute {
			t.Fatalf("got %v want 1h30m", h.D)
		}
	})

	t.Run("pointer-to-duration: allocates and sets", func(t *testing.T) {
		h := &holder{} // PD is nil
		if err := setLiteralDefault(field(h, "PD"), "250ms"); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if h.PD == nil {
			t.Fatalf("expected allocation for *time.Duration")
		}
		if *h.PD != 250*time.Millisecond {
			t.Fatalf("got %v want 250ms", *h.PD)
		}
	})

	t.Run("pointer-to-scalar: allocates and sets (int)", func(t *testing.T) {
		h := &holder{} // PI is nil
		if err := setLiteralDefault(field(h, "PI"), "9"); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if h.PI == nil || *h.PI != 9 {
			t.Fatalf("expected *int allocated with 9, got %#v", h.PI)
		}
	})

	t.Run("pointer-to-scalar non-zero: do not overwrite", func(t *testing.T) {
		h := &holder{PI: new(int)}
		*h.PI = 5
		if err := setLiteralDefault(field(h, "PI"), "10"); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if *h.PI != 5 {
			t.Fatalf("overwrote non-zero pointer-to-scalar: got %d", *h.PI)
		}
	})

	t.Run("pointer-to-complex (struct): do not auto-alloc, unsupported kind error", func(t *testing.T) {
		h := &holder{} // PSt is nil
		if err := setLiteralDefault(field(h, "PSt"), "ignored"); err == nil {
			t.Fatalf("expected unsupported kind error for pointer-to-struct with literal")
		}
		if h.PSt != nil {
			t.Fatalf("should not allocate *struct on literal default")
		}
	})

	t.Run("pointer-to-complex (slice): do not auto-alloc, unsupported kind error", func(t *testing.T) {
		h := &holder{} // no pointer field to slice, so simulate by taking address of SL field value (which is a slice)
		// Build a pointer-to-slice reflect.Value by taking address of the field
		f := field(h, "SL").Addr()
		if err := setLiteralDefault(f, "ignored"); err == nil {
			t.Fatalf("expected unsupported kind error for pointer-to-slice with literal")
		}
	})

	t.Run("unsupported kind: slice value", func(t *testing.T) {
		h := &holder{}
		if err := setLiteralDefault(field(h, "SL"), "ignored"); err == nil {
			t.Fatalf("expected error for unsupported kind slice")
		}
	})

	t.Run("unsupported kind: map value", func(t *testing.T) {
		h := &holder{}
		if err := setLiteralDefault(field(h, "MP"), "ignored"); err == nil {
			t.Fatalf("expected error for unsupported kind map")
		}
	})

	t.Run("unsupported kind: struct value", func(t *testing.T) {
		h := &holder{}
		if err := setLiteralDefault(field(h, "St"), "ignored"); err == nil {
			t.Fatalf("expected error for unsupported kind struct")
		}
	})

	t.Run("non-settable value: graceful no-op", func(t *testing.T) {
		// reflect.ValueOf on a non-addressable variable is not settable.
		v := reflect.ValueOf("")
		if err := setLiteralDefault(v, "x"); err != nil {
			t.Fatalf("unexpected error on non-settable value: %v", err)
		}
		// nothing to assert further; just ensuring no panic and no error
	})
	// TODO: add cases for other int/float kinds (e.g., int32, float32) by extending extend holder with those fields.
}
