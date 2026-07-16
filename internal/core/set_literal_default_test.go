package core

import (
	"reflect"
	"testing"
)

func TestSetLiteralValue_NonSettableValueIsNoop(t *testing.T) {
	value := reflect.ValueOf("")

	if err := setLiteralValue(value, "x", true); err != nil {
		t.Fatalf("setLiteralValue() error: %v", err)
	}
}
