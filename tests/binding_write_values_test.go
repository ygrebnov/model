package tests

/*
import (
	"errors"
	"testing"

	"github.com/ygrebnov/model"
	"github.com/ygrebnov/model/field"
	modelerrors "github.com/ygrebnov/model/pkg/errors"
)

type bindingValueSink map[string]any

func (m bindingValueSink) Set(field field.Field, value any) error {
	m[field.Path] = value
	return nil
}

type bindingErrValueSink struct {
	path string
	err  error
}

func (s bindingErrValueSink) Set(field field.Field, value any) error {
	if field.Path == s.path {
		return s.err
	}

	return nil
}

func TestBindingWriteValues(t *testing.T) {
	type nested struct {
		S string
	}
	type sample struct {
		S string
		P *nested
	}

	b, err := model.NewBinding[sample]()
	if err != nil {
		t.Fatalf("NewBinding returned unexpected error: %v", err)
	}

	obj := sample{
		S: "root",
		P: &nested{S: "nested"},
	}
	sink := bindingValueSink{}

	err = b.WriteValues(&obj, sink)
	if err != nil {
		t.Fatalf("WriteValues returned unexpected error: %v", err)
	}

	if got := sink["S"]; got != "root" {
		t.Fatalf("S = %#v, want %q", got, "root")
	}
	if got := sink["P"]; got != obj.P {
		t.Fatalf("P = %#v, want %#v", got, obj.P)
	}
	if got := sink["P.S"]; got != "nested" {
		t.Fatalf("P.S = %#v, want %q", got, "nested")
	}
}

func TestBindingWriteValues_SinkError(t *testing.T) {
	type sample struct {
		S string
	}

	b, err := model.NewBinding[sample]()
	if err != nil {
		t.Fatalf("NewBinding returned unexpected error: %v", err)
	}

	obj := sample{S: "x"}
	wantErr := errors.New("boom")

	err = b.WriteValues(&obj, bindingErrValueSink{path: "S", err: wantErr})
	if !errors.Is(err, wantErr) {
		t.Fatalf("WriteValues error = %v, want wrapped %v", err, wantErr)
	}
}

func TestBindingWriteValues_NilSink(t *testing.T) {
	type sample struct {
		S string
	}

	b, err := model.NewBinding[sample]()
	if err != nil {
		t.Fatalf("NewBinding returned unexpected error: %v", err)
	}

	obj := sample{}
	err = b.WriteValues(&obj, nil)
	if !errors.Is(err, modelerrors.ErrInvalidValue) {
		t.Fatalf("WriteValues error = %v, want %v", err, modelerrors.ErrInvalidValue)
	}
}
*/
