package validation

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/ygrebnov/errorc"

	errorsPkg "github.com/ygrebnov/model/errors"
)

// helper type for TestRegistry_get assignable interface scenario
// distinct Name to avoid conflict with other test types
type wrapGet struct{ v string }

func (w wrapGet) String() string { return w.v }

// Helper to fetch a builtin rule for tests after lazy-init refactor.
func builtinRuleForTest(t *testing.T, name string, typ reflect.Type) Rule {
	t.Helper()
	r, ok := lookupBuiltin(name, typ)
	if !ok {
		t.Fatalf("builtin rule %q for %s not found", name, typ)
	}
	return r
}

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

	// Additional overloads for stringRule (interface and int versions)
	interfaceOverloadForStringRule, _ := NewRule[interface{}]("stringRule", func(v interface{}, _ ...string) error { return errors.New("stringRule_iface") })
	intOverloadForStringRule, _ := NewRule[int]("stringRule", func(v int, _ ...string) error { return errors.New("stringRule_int") })
	// Duplicate pointer overload for pointerToInterfaceRule
	pointerToInterfaceRule2, _ := NewRule[*interface{}]("pointerToInterfaceRule", func(v *interface{}, _ ...string) error { return errors.New("pointerToInterfaceRule_dup") })
	// Duplicate interface overload for interfaceRule
	interfaceRule2, _ := NewRule[interface{}]("interfaceRule", func(v interface{}, _ ...string) error { return errors.New("interfaceRule_dup") })

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
			name:       "add duplicate overload rule for same type",
			rulesToAdd: []Rule{testRules["stringRule"], testRules["stringRule"]},
			expectedError: errorc.With(
				errorsPkg.ErrDuplicateOverloadRule,
				errorc.String(errorsPkg.ErrorFieldRuleName, "stringRule"),
				errorc.String(errorsPkg.ErrorFieldFieldType, "string"),
			),
		},
		{
			name:       "add rule with existing Name, but for different type",
			rulesToAdd: []Rule{testRules["stringRule"], interfaceOverloadForStringRule},
			expectedRules: map[string][]Rule{
				"stringRule": {testRules["stringRule"], interfaceOverloadForStringRule},
			},
		},
		{
			name:       "add multiple distinct overloads (string, interface, int)",
			rulesToAdd: []Rule{testRules["stringRule"], interfaceOverloadForStringRule, intOverloadForStringRule},
			expectedRules: map[string][]Rule{
				"stringRule": {testRules["stringRule"], interfaceOverloadForStringRule, intOverloadForStringRule},
			},
		},
		{
			name:       "short-circuit after duplicate (second add fails; third not applied)",
			rulesToAdd: []Rule{testRules["stringRule"], testRules["stringRule"], intOverloadForStringRule},
			expectedError: errorc.With(
				errorsPkg.ErrDuplicateOverloadRule,
				errorc.String(errorsPkg.ErrorFieldRuleName, "stringRule"),
				errorc.String(errorsPkg.ErrorFieldFieldType, "string"),
			),
		},
		{
			name:       "duplicate pointer overload",
			rulesToAdd: []Rule{testRules["pointerToInterfaceRule"], pointerToInterfaceRule2},
			expectedError: errorc.With(
				errorsPkg.ErrDuplicateOverloadRule,
				errorc.String(errorsPkg.ErrorFieldRuleName, "pointerToInterfaceRule"),
				errorc.String(errorsPkg.ErrorFieldFieldType, "*interface {}"),
			),
		},
		{
			name:       "duplicate interface overload",
			rulesToAdd: []Rule{testRules["interfaceRule"], interfaceRule2},
			expectedError: errorc.With(
				errorsPkg.ErrDuplicateOverloadRule,
				errorc.String(errorsPkg.ErrorFieldRuleName, "interfaceRule"),
				errorc.String(errorsPkg.ErrorFieldFieldType, "interface {}"),
			),
		},
		{
			name:          "nil rule",
			rulesToAdd:    []Rule{nil},
			expectedRules: map[string][]Rule{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reg := &registry{
				rules: make(map[string][]Rule),
			}
			var err error
			for _, r := range test.rulesToAdd {
				err = reg.Add(r)
				if err != nil {
					break
				}
			}
			if test.expectedError != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", test.expectedError)
				}
				if !errors.Is(err, errorsPkg.ErrDuplicateOverloadRule) {
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
					t.Fatalf("expected rule Name %s not found in registry", name)
				}
				if len(actualOverloads) != len(expectedOverloads) {
					t.Fatalf("for rule %s, expected %d overloads, got %d", name, len(expectedOverloads), len(actualOverloads))
				}

				// stable ordering by field type Name
				slices.SortFunc(actualOverloads, func(i, j Rule) int {
					return strings.Compare(i.GetFieldTypeName(), j.GetFieldTypeName())
				})
				slices.SortFunc(expectedOverloads, func(i, j Rule) int {
					return strings.Compare(i.GetFieldTypeName(), j.GetFieldTypeName())
				})

				for i, exp := range expectedOverloads {
					act := actualOverloads[i]
					if act.GetName() != exp.GetName() {
						t.Fatalf(
							"for rule %s overload %d, expected Name %s, got %s",
							name,
							i,
							exp.GetName(),
							act.GetName(),
						)
					}
					if act.GetFieldType() != exp.GetFieldType() {
						t.Fatalf(
							"for rule %s overload %d, expected type %s, got %s",
							name,
							i,
							exp.GetFieldTypeName(),
							act.GetFieldTypeName(),
						)
					}
				}
			}
		})
	}
}

func TestRegistry_get(t *testing.T) {
	testRules := getTestRules(t)

	defaultRegistry := func(t *testing.T) *registry {
		return &registry{
			rules: make(map[string][]Rule),
		}
	}

	cases := []struct { // rename internal for clarity
		name                  string
		setupRegistry         func(t *testing.T) *registry
		ruleName              string
		value                 reflect.Value
		expectedSentinelError error
		expectedError         error
		expectedRule          Rule
	}{
		{
			name:                  "invalid reflect.Value",
			setupRegistry:         defaultRegistry,
			ruleName:              "anything",
			value:                 reflect.Value{},
			expectedSentinelError: errorsPkg.ErrInvalidValue,
			expectedError: errorc.With(
				errorsPkg.ErrInvalidValue,
				errorc.String(errorsPkg.ErrorFieldRuleName, "anything"),
			),
		},
		{
			name:                  "rule not found (no custom, no builtin)",
			setupRegistry:         defaultRegistry,
			ruleName:              "doesNotExist",
			value:                 reflect.ValueOf(123),
			expectedSentinelError: errorsPkg.ErrRuleNotFound,
			expectedError: errorc.With(
				errorsPkg.ErrRuleNotFound,
				errorc.String(errorsPkg.ErrorFieldRuleName, "doesNotExist"),
			),
		},
		{
			name:          "builtin fallback only (string email)",
			setupRegistry: defaultRegistry,
			ruleName:      "email",
			value:         reflect.ValueOf(""),
			expectedRule:  builtinRuleForTest(t, "email", reflect.TypeOf("")),
		},
		{
			name: "builtin fallback when empty slice present",
			setupRegistry: func(t *testing.T) *registry {
				r := &registry{
					rules: make(map[string][]Rule),
				}
				r.rules["email"] = []Rule{}
				return r
			},
			ruleName:     "email",
			value:        reflect.ValueOf(""),
			expectedRule: builtinRuleForTest(t, "email", reflect.TypeOf("")),
		},
		{
			name: "exact match single overload",
			setupRegistry: func(t *testing.T) *registry {
				r := &registry{
					rules: make(map[string][]Rule),
				}
				r.rules["singleOverload"] = []Rule{testRules["stringRule"]}
				return r
			},
			ruleName:     "singleOverload",
			value:        reflect.ValueOf("hi"),
			expectedRule: testRules["stringRule"],
		},
		{
			name: "assignable interface match (no exact)",
			setupRegistry: func(t *testing.T) *registry {
				r := &registry{
					rules: make(map[string][]Rule),
				}
				r.rules["assignableInterface"] = []Rule{testRules["stringerInterfaceRule"]}
				return r
			},
			ruleName:     "assignableInterface",
			value:        reflect.ValueOf(wrapGet{v: "W"}),
			expectedRule: testRules["stringerInterfaceRule"],
		},
		{
			name: "exact preferred over assignable (both registered)",
			setupRegistry: func(t *testing.T) *registry {
				r := &registry{
					rules: make(map[string][]Rule),
				}
				r.rules["exactOverAssignable"] = []Rule{testRules["stringerInterfaceRule"], testRules["stringRule"]}
				return r
			},
			ruleName:     "exactOverAssignable",
			value:        reflect.ValueOf("ZZ"),
			expectedRule: testRules["stringRule"],
		},
		{
			name: "no overload for value type -> available types list",
			setupRegistry: func(t *testing.T) *registry {
				r := &registry{
					rules: make(map[string][]Rule),
				}
				r.rules["noOverload"] = []Rule{testRules["stringRule"], testRules["intRule"]}
				return r
			},
			ruleName:              "noOverload",
			value:                 reflect.ValueOf(3.14),
			expectedSentinelError: errorsPkg.ErrRuleOverloadNotFound,
			expectedError: errorc.With(
				errorsPkg.ErrRuleOverloadNotFound,
				errorc.String(errorsPkg.ErrorFieldRuleName, "noOverload"),
				errorc.String(errorsPkg.ErrorFieldValueType, "float64"),
				errorc.String(errorsPkg.ErrorFieldAvailableTypes, "int, string"),
			),
		},
		{
			name: "ambiguous duplicates (manually inserted unreachable path)",
			setupRegistry: func(t *testing.T) *registry {
				r := &registry{
					rules: make(map[string][]Rule),
				}
				r.rules["ambiguousDuplicates"] = []Rule{testRules["stringRule"], testRules["stringRule"]}
				return r
			},
			ruleName:              "ambiguousDuplicates",
			value:                 reflect.ValueOf("x"),
			expectedSentinelError: errorsPkg.ErrAmbiguousRule,
			expectedError: errorc.With(
				errorsPkg.ErrAmbiguousRule,
				errorc.String(errorsPkg.ErrorFieldRuleName, "ambiguousDuplicates"),
				errorc.String(errorsPkg.ErrorFieldValueType, "string"),
			),
		},
	}

	for _, tc := range cases {
		// capture
		t.Run(tc.name, func(t *testing.T) {
			r := tc.setupRegistry(t)
			rule, err := r.Get(tc.ruleName, tc.value)
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
				if rule.GetName() != tc.expectedRule.GetName() {
					t.Fatalf(
						"expected rule Name %s, got %s",
						tc.expectedRule.GetName(),
						rule.GetName(),
					)
				}
				if rule.GetFieldType() != tc.expectedRule.GetFieldType() {
					t.Fatalf(
						"expected rule field type %s, got %s",
						tc.expectedRule.GetFieldType(),
						rule.GetFieldType(),
					)
				}
			}
		})
	}
}
