package core

import (
	"errors"
	"reflect"
	"testing"

	fieldPkg "github.com/ygrebnov/model/field"
	modelerrors "github.com/ygrebnov/model/pkg/errors"
)

type mapValueSource map[string]any

func (m mapValueSource) Get(f fieldPkg.Field) (any, bool, error) {
	value, ok := m[f.Path]
	return value, ok, nil
}

func TestApplyValuesStruct(t *testing.T) {
	type nested struct {
		S string
	}
	type sample struct {
		S  string
		I  int
		PS *string
		P  *nested
	}

	obj := &sample{}
	rv := reflect.ValueOf(obj).Elem()

	err := newService(obj).ApplyValuesStruct(rv, mapValueSource{
		"S":   "root",
		"I":   int8(7),
		"PS":  "ptr",
		"P.S": "nested",
	})
	if err != nil {
		t.Fatalf("ApplyValuesStruct returned unexpected error: %v", err)
	}

	if obj.S != "root" {
		t.Fatalf("S = %q, want %q", obj.S, "root")
	}
	if obj.I != 7 {
		t.Fatalf("I = %d, want %d", obj.I, 7)
	}
	if obj.PS == nil || *obj.PS != "ptr" {
		t.Fatalf("PS = %#v, want pointer to %q", obj.PS, "ptr")
	}
	if obj.P == nil || obj.P.S != "nested" {
		t.Fatalf("P = %#v, want nested value", obj.P)
	}
}

func TestApplyValuesStruct_TypeMismatch(t *testing.T) {
	type sample struct {
		I int
	}

	obj := &sample{}
	rv := reflect.ValueOf(obj).Elem()

	err := newService(obj).ApplyValuesStruct(rv, mapValueSource{
		"I": "bad",
	})
	if !errors.Is(err, modelerrors.ErrTypeMismatch) {
		t.Fatalf("ApplyValuesStruct error = %v, want %v", err, modelerrors.ErrTypeMismatch)
	}
}
