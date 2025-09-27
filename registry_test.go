package model

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/ygrebnov/errorc"
)

// helper type for TestRegistry_get assignable interface scenario
// distinct name to avoid conflict with other test types
type wrapGet struct{ v string }

func (w wrapGet) String() string { return w.v }

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

func TestRegistry_get(t *testing.T) {
	stringRule, err := NewRule[string]("sx", func(v string, _ ...string) error { return fmt.Errorf("sx:%s", v) })
	if err != nil {
		t.Fatalf("NewRule stringRule error: %v", err)
	}
	intRule, err := NewRule[int]("sx", func(v int, _ ...string) error { return fmt.Errorf("ix:%d", v) })
	if err != nil {
		t.Fatalf("NewRule intRule error: %v", err)
	}
	// interface rule for assignable path
	type stringer interface{ String() string }
	ifaceRule, err := NewRule[stringer]("ifaceOnly", func(s stringer, _ ...string) error { return fmt.Errorf("iface:%s", s.String()) })
	if err != nil {
		t.Fatalf("NewRule ifaceRule error: %v", err)
	}

	tests := []struct {
		name       string
		setup      func() *registry
		ruleName   string
		value      reflect.Value
		wantErrSub string
		wantRuleFn string // substring expected from rule fn when invoked
	}{
		{
			name:       "invalid reflect.Value",
			setup:      func() *registry { return newRegistry() },
			ruleName:   "anything",
			value:      reflect.Value{},
			wantErrSub: "invalid value",
		},
		{
			name:       "rule not found (no custom, no builtin)",
			setup:      func() *registry { return newRegistry() },
			ruleName:   "doesNotExist",
			value:      reflect.ValueOf(123),
			wantErrSub: "rule not found",
		},
		{
			name:       "builtin fallback only (string nonempty)",
			setup:      func() *registry { return newRegistry() },
			ruleName:   "nonempty",
			value:      reflect.ValueOf(""),
			wantRuleFn: "must not be empty", // builtin error substring
		},
		{
			name: "builtin fallback when empty slice present",
			setup: func() *registry {
				r := newRegistry()
				// simulate name present with empty overload slice
				r.rules["nonempty"] = []Rule{}
				return r
			},
			ruleName:   "nonempty",
			value:      reflect.ValueOf(""),
			wantRuleFn: "must not be empty",
		},
		{
			name: "exact match single overload",
			setup: func() *registry {
				r := newRegistry()
				_ = r.add(stringRule)
				return r
			},
			ruleName:   "sx",
			value:      reflect.ValueOf("hi"),
			wantRuleFn: "sx:hi",
		},
		{
			name:       "assignable interface match (no exact)",
			setup:      func() *registry { r := newRegistry(); _ = r.add(ifaceRule); return r },
			ruleName:   "ifaceOnly",
			value:      reflect.ValueOf(wrapGet{v: "W"}),
			wantRuleFn: "iface:W",
		},
		{
			name: "exact preferred over assignable (both registered)",
			setup: func() *registry {
				r := newRegistry()
				_ = r.add(ifaceRule)
				// add exact string rule with same name
				strExact, _ := NewRule[string]("ifaceOnly", func(s string, _ ...string) error { return fmt.Errorf("exact:%s", s) })
				_ = r.add(strExact)
				return r
			},
			ruleName:   "ifaceOnly",
			value:      reflect.ValueOf("ZZ"),
			wantRuleFn: "exact:ZZ",
		},
		{
			name:       "no overload for value type -> available types list",
			setup:      func() *registry { r := newRegistry(); _ = r.add(stringRule); _ = r.add(intRule); return r },
			ruleName:   "sx",
			value:      reflect.ValueOf(3.14), // float64 -> none matches
			wantErrSub: "available_types: int, string",
		},
		{
			name: "ambiguous duplicates (manually inserted unreachable path)",
			setup: func() *registry {
				r := newRegistry()
				// force two exact duplicates bypassing add's guard
				r.rules["dup"] = []Rule{stringRule, stringRule}
				return r
			},
			ruleName:   "dup",
			value:      reflect.ValueOf("x"),
			wantErrSub: "ambiguous",
		},
	}

	for _, tc := range tests {
		// capture
		t.Run(tc.name, func(t *testing.T) {
			r := tc.setup()
			rule, err := r.get(tc.ruleName, tc.value)
			if tc.wantErrSub != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErrSub) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErrSub, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// invoke rule fn to confirm identity
			if tc.wantRuleFn != "" {
				err2 := rule.getValidationFn()(tc.value)
				if err2 == nil || !strings.Contains(err2.Error(), tc.wantRuleFn) {
					// builtin rules may succeed depending on input; ensure substring
					if err2 == nil || !strings.Contains(err2.Error(), tc.wantRuleFn) {
						// retry only for builtin nonempty with non-empty value? Not needed here
						// Fail
						// Provide context
						t.Fatalf("expected rule fn error containing %q, got %v", tc.wantRuleFn, err2)
					}
				}
			}
		})
	}
}
