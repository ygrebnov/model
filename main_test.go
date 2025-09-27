package model

import (
	"errors"
	"testing"
)

type a struct{}

func getTestRules(t *testing.T) map[string]Rule {
	stringRule, err := NewRule(
		"stringRule",
		func(v string, _ ...string) error { return errors.New("stringRule") },
	)
	if err != nil {
		t.Fatalf("NewRule error: %v", err)
	}
	intRule, err := NewRule(
		"intRule",
		func(v int, _ ...string) error { return errors.New("intRule") },
	)
	if err != nil {
		t.Fatalf("NewRule error: %v", err)
	}
	floatRule, err := NewRule(
		"floatRule",
		func(v float64, _ ...string) error { return errors.New("floatRule") },
	)
	if err != nil {
		t.Fatalf("NewRule error: %v", err)
	}
	interfaceRule, err := NewRule(
		"interfaceRule",
		func(v interface{}, _ ...string) error { return errors.New("interfaceRule") },
	)
	if err != nil {
		t.Fatalf("NewRule error: %v", err)
	}
	structARule, err := NewRule(
		"structARule",
		func(v a, _ ...string) error { return errors.New("structARule") },
	)
	if err != nil {
		t.Fatalf("NewRule error: %v", err)
	}
	structRule, err := NewRule(
		"structRule",
		func(v struct{}, _ ...string) error { return errors.New("structRule") },
	)
	if err != nil {
		t.Fatalf("NewRule error: %v", err)
	}
	pointerToInterfaceRule, err := NewRule(
		"pointerToInterfaceRule",
		func(v *interface{}, _ ...string) error { return errors.New("pointerToInterfaceRule") },
	)
	if err != nil {
		t.Fatalf("NewRule error: %v", err)
	}
	return map[string]Rule{
		"stringRule":             stringRule,
		"intRule":                intRule,
		"floatRule":              floatRule,
		"interfaceRule":          interfaceRule,
		"structARule":            structARule,
		"structRule":             structRule,
		"pointerToInterfaceRule": pointerToInterfaceRule,
	}
}
