package model

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/ygrebnov/errorc"
)

func TestGetFieldTypesNames(t *testing.T) {
	testRules := getTestRules(t)

	tests := []struct {
		name     string
		rules    []Rule
		expected []string
	}{
		{
			name:     "single rule",
			rules:    []Rule{testRules["stringRule"]},
			expected: []string{"string"},
		},
		{
			name: "multiple rules of different types, unsorted",
			rules: []Rule{
				testRules["stringRule"],
				testRules["intRule"],
				testRules["floatRule"],
				testRules["interfaceRule"],
				testRules["structARule"],
				testRules["structRule"],
				testRules["pointerToInterfaceRule"],
			},
			expected: []string{
				"*interface {}",
				"float64",
				"int",
				"interface {}",
				"model.a",
				"string",
				"struct {}",
			},
		},
		{
			name:     "empty rules",
			rules:    []Rule{},
			expected: []string{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := getFieldTypesNames(test.rules)
			if len(actual) != len(test.expected) {
				t.Fatalf("expected %d types, got %d", len(test.expected), len(actual))
			}
			for i, exp := range test.expected {
				if actual[i] != exp {
					t.Fatalf("at index %d, expected %s, got %s", i, exp, actual[i])
				}
			}
		})
	}
}

func TestRegistry_add(t *testing.T) {
	testRules := getTestRules(t)

	stringRule2, err := NewRule(
		"stringRule",
		func(v interface{}, _ ...string) error { return errors.New("stringRule2") },
	)
	if err != nil {
		t.Fatalf("NewRule error: %v", err)
	}
	intOverloadForStringRule, err := NewRule(
		"stringRule",
		func(v int, _ ...string) error { return errors.New("stringRule_intOverload") },
	)
	if err != nil {
		t.Fatalf("NewRule intOverloadForStringRule error: %v", err)
	}
	pointerToInterfaceRule2, err := NewRule(
		"pointerToInterfaceRule",
		func(v *interface{}, _ ...string) error { return errors.New("pointerToInterfaceRule2") },
	)
	if err != nil {
		t.Fatalf("NewRule pointerToInterfaceRule2 error: %v", err)
	}
	interfaceRule2, err := NewRule(
		"interfaceRule",
		func(v interface{}, _ ...string) error { return errors.New("interfaceRule2") },
	)
	if err != nil {
		t.Fatalf("NewRule interfaceRule2 error: %v", err)
	}

	tests := []struct {
		name          string
		rulesToAdd    []Rule
		expectedError error
		expectedRules map[string][]Rule
	}{
		{
			name:       "add single rule",
			rulesToAdd: []Rule{testRules["stringRule"]},
			expectedRules: map[string][]Rule{
				"stringRule": {testRules["stringRule"]},
			},
		},
		{
			name: "add duplicate overload rule for same type",
			rulesToAdd: []Rule{
				testRules["stringRule"],
				testRules["stringRule"],
			},
			expectedError: errorc.With(
				ErrDuplicateOverloadRule,
				errorc.Field("rule_name", "stringRule"),
				errorc.Field("field_type", "string"),
			),
		},
		{
			name: "add rule with existing name, but for different type",
			rulesToAdd: []Rule{
				testRules["stringRule"],
				stringRule2, // interface{} overload
			},
			expectedRules: map[string][]Rule{
				"stringRule": {testRules["stringRule"], stringRule2},
			},
		},
		{
			name: "add multiple distinct overloads (string, interface, int)",
			rulesToAdd: []Rule{
				testRules["stringRule"],
				stringRule2,
				intOverloadForStringRule,
			},
			expectedRules: map[string][]Rule{
				"stringRule": {testRules["stringRule"], stringRule2, intOverloadForStringRule},
			},
		},
		{
			name: "short-circuit after duplicate (second add fails; third not applied)",
			rulesToAdd: []Rule{
				testRules["stringRule"],  // ok
				testRules["stringRule"],  // duplicate -> error
				intOverloadForStringRule, // must NOT be added
			},
			expectedError: errorc.With(
				ErrDuplicateOverloadRule,
				errorc.Field("rule_name", "stringRule"),
				errorc.Field("field_type", "string"),
			),
		},
		{
			name: "duplicate pointer overload",
			rulesToAdd: []Rule{
				testRules["pointerToInterfaceRule"],
				pointerToInterfaceRule2, // duplicate *interface{}
			},
			expectedError: errorc.With(
				ErrDuplicateOverloadRule,
				errorc.Field("rule_name", "pointerToInterfaceRule"),
				errorc.Field("field_type", "*interface {}"),
			),
		},
		{
			name: "duplicate interface overload",
			rulesToAdd: []Rule{
				testRules["interfaceRule"],
				interfaceRule2,
			},
			expectedError: errorc.With(
				ErrDuplicateOverloadRule,
				errorc.Field("rule_name", "interfaceRule"),
				errorc.Field("field_type", "interface {}"),
			),
		},
		{
			name:          "nil rule",
			rulesToAdd:    []Rule{nil},
			expectedRules: map[string][]Rule{},
		},
	}

	for _, test := range tests {
		// capture range var
		t.Run(test.name, func(t *testing.T) {
			reg := newRegistry()
			var err error
			for _, r := range test.rulesToAdd {
				err = reg.add(r)
				if err != nil {
					break
				}
			}
			if test.expectedError != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", test.expectedError)
				}
				if !errors.Is(err, ErrDuplicateOverloadRule) {
					t.Fatalf("expected ErrDuplicateOverloadRule, got %T", err)
				}
				if err.Error() != test.expectedError.Error() {
					t.Fatalf("expected error %v, got %v", test.expectedError, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Validate internal registry state
			if len(reg.rules) != len(test.expectedRules) {
				t.Fatalf("expected %d rule entries, got %d", len(test.expectedRules), len(reg.rules))
			}
			for name, expectedOverloads := range test.expectedRules {
				actualOverloads, exists := reg.rules[name]
				if !exists {
					t.Fatalf("expected rule name %s not found in registry", name)
				}
				if len(actualOverloads) != len(expectedOverloads) {
					t.Fatalf("for rule %s, expected %d overloads, got %d", name, len(expectedOverloads), len(actualOverloads))
				}

				// stable ordering by field type name
				slices.SortFunc(actualOverloads, func(i, j Rule) int {
					return strings.Compare(i.getFieldTypeName(), j.getFieldTypeName())
				})
				slices.SortFunc(expectedOverloads, func(i, j Rule) int {
					return strings.Compare(i.getFieldTypeName(), j.getFieldTypeName())
				})

				for i, exp := range expectedOverloads {
					act := actualOverloads[i]
					if act.getName() != exp.getName() {
						t.Fatalf("for rule %s overload %d, expected name %s, got %s", name, i, exp.getName(), act.getName())
					}
					if act.getFieldType() != exp.getFieldType() {
						t.Fatalf("for rule %s overload %d, expected type %s, got %s", name, i, exp.getFieldTypeName(), act.getFieldTypeName())
					}
				}
			}
		})
	}
}
