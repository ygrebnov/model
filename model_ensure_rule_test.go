package model

import (
	"context"
	"testing"
)

func TestWithValidation_BuiltinsRemainValid_NoError(t *testing.T) {
	type Obj struct{ S string }
	obj := Obj{}
	if _, err := New(&obj, WithValidation[Obj](context.Background())); err != nil {
		t.Fatalf("WithValidation should not error for valid builtins, got: %v", err)
	}
}
