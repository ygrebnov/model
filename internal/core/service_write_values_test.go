package core

import (
	"errors"
	"reflect"
	"testing"

	fieldPkg "github.com/ygrebnov/model/field"
	modelerrors "github.com/ygrebnov/model/pkg/errors"
)

type mapValueSink map[string]any

func (m mapValueSink) Set(field fieldPkg.Field, value any) error {
	m[field.Path] = value
	return nil
}

type errValueSink struct {
	path string
	err  error
}

func (s errValueSink) Set(field fieldPkg.Field, value any) error {
	if field.Path == s.path {
		return s.err
	}

	return nil
}

func TestWriteValuesStruct(t *testing.T) {
	type nested struct {
		S string
	}
	type sample struct {
		S  string
		I  int
		PS *string
		P  *nested
	}

	ptrValue := "ptr"
	obj := &sample{
		S:  "root",
		I:  7,
		PS: &ptrValue,
		P:  &nested{S: "nested"},
	}
	rv := reflect.ValueOf(obj).Elem()
	sink := mapValueSink{}

	err := newService(obj).WriteValuesStruct(rv, sink)
	if err != nil {
		t.Fatalf("WriteValuesStruct returned unexpected error: %v", err)
	}

	if got := sink["S"]; got != "root" {
		t.Fatalf("S = %#v, want %q", got, "root")
	}
	if got := sink["I"]; got != 7 {
		t.Fatalf("I = %#v, want %d", got, 7)
	}
	if got := sink["PS"]; got != obj.PS {
		t.Fatalf("PS = %#v, want %#v", got, obj.PS)
	}
	if got := sink["P"]; got != obj.P {
		t.Fatalf("P = %#v, want %#v", got, obj.P)
	}
	if got := sink["P.S"]; got != "nested" {
		t.Fatalf("P.S = %#v, want %q", got, "nested")
	}
}

func TestWriteValuesStruct_NilNestedPointerSkipsChildren(t *testing.T) {
	type nested struct {
		S string
	}
	type sample struct {
		P *nested
	}

	obj := &sample{}
	rv := reflect.ValueOf(obj).Elem()
	sink := mapValueSink{}

	err := newService(obj).WriteValuesStruct(rv, sink)
	if err != nil {
		t.Fatalf("WriteValuesStruct returned unexpected error: %v", err)
	}

	got, ok := sink["P"]
	if !ok {
		t.Fatal("missing P")
	}
	if got != obj.P {
		t.Fatalf("P = %#v, want %#v", got, obj.P)
	}
	if _, ok := sink["P.S"]; ok {
		t.Fatal("unexpected nested value for nil pointer")
	}
}

func TestWriteValuesStruct_SinkError(t *testing.T) {
	type sample struct {
		S string
	}

	obj := &sample{S: "x"}
	rv := reflect.ValueOf(obj).Elem()
	wantErr := errors.New("boom")

	err := newService(obj).WriteValuesStruct(rv, errValueSink{path: "S", err: wantErr})
	if !errors.Is(err, wantErr) {
		t.Fatalf("WriteValuesStruct error = %v, want wrapped %v", err, wantErr)
	}
}

func TestWriteValuesStruct_NilSink(t *testing.T) {
	type sample struct {
		S string
	}

	obj := &sample{}
	rv := reflect.ValueOf(obj).Elem()

	err := newService(obj).WriteValuesStruct(rv, nil)
	if !errors.Is(err, modelerrors.ErrInvalidValue) {
		t.Fatalf("WriteValuesStruct error = %v, want %v", err, modelerrors.ErrInvalidValue)
	}
}
