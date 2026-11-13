package core

import (
	"errors"
	"reflect"
	"testing"

	modelerrors "github.com/ygrebnov/model/pkg/errors"
)

func TestSetLiteralValue_NonSettableValueIsNoop(t *testing.T) {
	value := reflect.ValueOf("")

	if err := setLiteralValue(value, "x", true); err != nil {
		t.Fatalf("setLiteralValue() error: %v", err)
	}
}

func TestSetLiteralValue_RejectsOutOfRangeNumbers(t *testing.T) {
	tests := []struct {
		name    string
		typ     reflect.Type
		literal string
	}{
		{"positive int8", reflect.TypeFor[int8](), "128"},
		{"negative int8", reflect.TypeFor[int8](), "-129"},
		{"uint8", reflect.TypeFor[uint8](), "256"},
		{"float32", reflect.TypeFor[float32](), "1e40"},
		{"complex64", reflect.TypeFor[complex64](), "1e40+1i"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := reflect.New(test.typ).Elem()
			err := setLiteralValue(value, test.literal, true)
			if !errors.Is(err, modelerrors.ErrNumericValueOutOfRange) {
				t.Fatalf("setLiteralValue() error = %v, want out-of-range error", err)
			}
			if !value.IsZero() {
				t.Fatalf("value = %v, want zero after rejected literal", value)
			}
		})
	}
}
