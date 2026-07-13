package tests

/*
import (
	"errors"
	"testing"

	"github.com/ygrebnov/model"
	"github.com/ygrebnov/model/field"
	modelerrors "github.com/ygrebnov/model/pkg/errors"
)

type bindingValueSource map[string]any

func (m bindingValueSource) Get(field field.Field) (any, bool, error) {
	value, ok := m[field.Path]
	return value, ok, nil
}

func TestBindingApplyValues(t *testing.T) {
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

	obj := sample{}
	err = b.ApplyValues(&obj, bindingValueSource{
		"S":   "root",
		"P.S": "nested",
	})
	if err != nil {
		t.Fatalf("ApplyValues returned unexpected error: %v", err)
	}

	if obj.S != "root" {
		t.Fatalf("S = %q, want %q", obj.S, "root")
	}
	if obj.P == nil || obj.P.S != "nested" {
		t.Fatalf("P = %#v, want nested value", obj.P)
	}
}

func TestBindingApplyValues_TypeMismatch(t *testing.T) {
	type sample struct {
		I int
	}

	b, err := model.NewBinding[sample]()
	if err != nil {
		t.Fatalf("NewBinding returned unexpected error: %v", err)
	}

	obj := sample{}
	err = b.ApplyValues(&obj, bindingValueSource{
		"I": "bad",
	})
	if !errors.Is(err, modelerrors.ErrTypeMismatch) {
		t.Fatalf("ApplyValues error = %v, want %v", err, modelerrors.ErrTypeMismatch)
	}
}
*/
