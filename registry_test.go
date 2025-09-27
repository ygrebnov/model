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
				testRules["interfaceOverloadForStringRule"],
			},
			expectedRules: map[string][]Rule{
				"stringRule": {testRules["stringRule"], testRules["interfaceOverloadForStringRule"]},
			},
		},
		{
			name: "add multiple distinct overloads (string, interface, int)",
			rulesToAdd: []Rule{
				testRules["stringRule"],
				testRules["interfaceOverloadForStringRule"],
				testRules["intOverloadForStringRule"],
			},
			expectedRules: map[string][]Rule{
				"stringRule": {
					testRules["stringRule"],
					testRules["interfaceOverloadForStringRule"],
					testRules["intOverloadForStringRule"],
				},
			},
		},
		{
			name: "short-circuit after duplicate (second add fails; third not applied)",
			rulesToAdd: []Rule{
				testRules["stringRule"],               // ok
				testRules["stringRule"],               // duplicate -> error
				testRules["intOverloadForStringRule"], // must NOT be added
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
				testRules["pointerToInterfaceRule2"], // duplicate *interface{}
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
				testRules["interfaceRule2"],
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
	// testRules := getTestRules(t)

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

	defaultRegistry := func(t *testing.T) *registry { return newRegistry() }

	tests := []struct {
		name                  string
		setupRegistry         func(t *testing.T) *registry
		ruleName              string
		value                 reflect.Value
		expectedSentinelError error
		expectedError         error
		expectedRule          Rule

		wantErrSub string
		wantRuleFn string // substring expected from rule fn when invoked
	}{
		{
			name:                  "invalid reflect.Value",
			setupRegistry:         defaultRegistry,
			ruleName:              "anything",
			value:                 reflect.Value{},
			wantErrSub:            "invalid value",
			expectedSentinelError: ErrInvalidValue,
			expectedError:         errorc.With(ErrInvalidValue, errorc.Field("rule_name", "anything")),
		},
		{
			name:                  "rule not found (no custom, no builtin)",
			setupRegistry:         defaultRegistry,
			ruleName:              "doesNotExist",
			value:                 reflect.ValueOf(123),
			wantErrSub:            "rule not found",
			expectedSentinelError: ErrRuleNotFound,
			expectedError:         errorc.With(ErrRuleNotFound, errorc.Field("rule_name", "doesNotExist")),
		},
		{
			name:          "builtin fallback only (string nonempty)",
			setupRegistry: defaultRegistry,
			ruleName:      "nonempty",
			value:         reflect.ValueOf(""),
			wantRuleFn:    "must not be empty", // builtin error substring
			expectedRule:  BuiltinStringRules()[0],
		},
		{
			name: "builtin fallback when empty slice present",
			setupRegistry: func(t *testing.T) *registry {
				r := newRegistry()
				// simulate name present with empty overload slice
				r.rules["nonempty"] = []Rule{}
				return r
			},
			ruleName:     "nonempty",
			value:        reflect.ValueOf(""),
			wantRuleFn:   "must not be empty",
			expectedRule: BuiltinStringRules()[0],
		},
		{
			name: "exact match single overload",
			setupRegistry: func(t *testing.T) *registry {
				r := newRegistry()
				e := r.add(stringRule)
				if e != nil {
					t.Fatalf("add stringRule error: %v", e)
				}
				return r
			},
			ruleName:   "sx",
			value:      reflect.ValueOf("hi"),
			wantRuleFn: "sx:hi",
		},
		{
			name: "assignable interface match (no exact)",
			setupRegistry: func(t *testing.T) *registry {
				r := newRegistry()
				e := r.add(ifaceRule)
				if e != nil {
					t.Fatalf("add ifaceRule error: %v", e)
				}
				return r
			},
			ruleName:   "ifaceOnly",
			value:      reflect.ValueOf(wrapGet{v: "W"}),
			wantRuleFn: "iface:W",
		},
		{
			name: "exact preferred over assignable (both registered)",
			setupRegistry: func(t *testing.T) *registry {
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
			name:          "no overload for value type -> available types list",
			setupRegistry: func(t *testing.T) *registry { r := newRegistry(); _ = r.add(stringRule); _ = r.add(intRule); return r },
			ruleName:      "sx",
			value:         reflect.ValueOf(3.14), // float64 -> none matches
			wantErrSub:    "available_types: int, string",
		},
		{
			name: "ambiguous duplicates (manually inserted unreachable path)",
			setupRegistry: func(t *testing.T) *registry {
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
			r := tc.setupRegistry(t)
			rule, err := r.get(tc.ruleName, tc.value)
			if tc.expectedError != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tc.expectedError)
				}
				if !errors.Is(err, tc.expectedSentinelError) {
					t.Fatalf("expected error type %T, got %T", tc.expectedSentinelError, err)
				}
				if err.Error() != tc.expectedError.Error() {
					t.Fatalf("expected error %v, got %v", tc.expectedError, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if rule == nil {
					t.Fatalf("expected rule, got nil")
				}
				if rule.getName() != tc.expectedRule.getName() {
					t.Fatalf(
						"expected rule name %s, got %s",
						tc.expectedRule.getName(),
						rule.getName(),
					)
				}
				if rule.getFieldType() != tc.expectedRule.getFieldType() {
					t.Fatalf(
						"expected rule field type %s, got %s",
						tc.expectedRule.getFieldType(),
						rule.getFieldType(),
					)
				}
			}

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
