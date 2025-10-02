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
	intOverloadForStringRule, err := NewRule(
		"stringRule",
		func(v int, _ ...string) error { return errors.New("intOverloadForStringRule") },
	)
	if err != nil {
		t.Fatalf("NewRule error: %v", err)
	}
	interfaceOverloadForStringRule, err := NewRule(
		"stringRule",
		func(v interface{}, _ ...string) error { return errors.New("interfaceOverloadForStringRule") },
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
	interfaceRule2, err := NewRule(
		"interfaceRule",
		func(v interface{}, _ ...string) error { return errors.New("interfaceRule2") },
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
	pointerToInterfaceRule2, err := NewRule(
		"pointerToInterfaceRule",
		func(v *interface{}, _ ...string) error { return errors.New("pointerToInterfaceRule2") },
	)
	if err != nil {
		t.Fatalf("NewRule error: %v", err)
	}
	// interface rule for assignable path
	type stringer interface{ String() string }
	stringerInterfaceRule, err := NewRule[stringer](
		"stringerInterfaceRule",
		func(s stringer, _ ...string) error { return errors.New("stringerInterfaceRule") })
	if err != nil {
		t.Fatalf("NewRule error: %v", err)
	}

	return map[string]Rule{
		"stringRule":                     stringRule,
		"intOverloadForStringRule":       intOverloadForStringRule,
		"interfaceOverloadForStringRule": interfaceOverloadForStringRule,
		"intRule":                        intRule,
		"floatRule":                      floatRule,
		"interfaceRule":                  interfaceRule,
		"interfaceRule2":                 interfaceRule2,
		"structARule":                    structARule,
		"structRule":                     structRule,
		"pointerToInterfaceRule":         pointerToInterfaceRule,
		"pointerToInterfaceRule2":        pointerToInterfaceRule2,
		"stringerInterfaceRule":          stringerInterfaceRule,
	}
}
