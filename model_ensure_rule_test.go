package model

import "testing"

func TestWithValidation_BuiltinsRemainValid_NoError(t *testing.T) {
	type Obj struct{ S string }
	obj := Obj{}
	if _, err := New(&obj, WithValidation[Obj]()); err != nil {
		t.Fatalf("WithValidation should not error for valid builtins, got: %v", err)
	}
}
