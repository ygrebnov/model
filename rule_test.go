package model

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// helper types for interface assignability tests

type myStringerImpl struct{ s string }

func (m myStringerImpl) String() string { return m.s }

type structSimple struct{ A int }

func TestNewRule(t *testing.T) {
	tests := []struct {
		name     string
		ruleName string
		fn       any // provided to newRule via type assertion inside test
		assert   func(r *validationRule, err error)
	}{
		{
			name:     "empty name returns error",
			ruleName: "",
			fn:       func(int, ...string) error { return nil },
			assert: func(r *validationRule, err error) {
				if !errors.Is(err, ErrInvalidRule) {
					t.Fatalf("expected ErrInvalidRule, got %v", err)
				}
				if r != nil {
					t.Fatalf("expected nil rule on error")
				}
			},
		},
		{
			name:     "nil function returns error",
			ruleName: "r1",
			fn:       nil,
			assert: func(r *validationRule, err error) {
				if !errors.Is(err, ErrInvalidRule) {
					t.Fatalf("expected ErrInvalidRule, got %v", err)
				}
				if r != nil {
					t.Fatalf("expected nil rule on error")
				}
			},
		},
		{
			name:     "primitive int rule",
			ruleName: "intRule",
			fn:       func(int, ...string) error { return nil },
			assert: func(r *validationRule, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if r.getName() != "intRule" {
					t.Fatalf("unexpected name %s", r.getName())
				}
				if r.getFieldType() != reflect.TypeOf(int(0)) {
					t.Fatalf("unexpected field type %s", r.getFieldType())
				}
			},
		},
		{
			name:     "interface rule fmt.Stringer",
			ruleName: "stringer",
			fn:       func(fmt.Stringer, ...string) error { return nil },
			assert: func(r *validationRule, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if r.getFieldType().Kind() != reflect.Interface {
					t.Fatalf("expected interface kind")
				}
			},
		},
		{
			name:     "pointer type rule",
			ruleName: "ptrInt",
			fn:       func(*int, ...string) error { return nil },
			assert: func(r *validationRule, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if r.getFieldType().Kind() != reflect.Ptr {
					t.Fatalf("expected pointer kind")
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// build rule according to fn's inferred generic parameter
			var (
				rule *validationRule
				err  error
			)
			switch f := tt.fn.(type) {
			case func(int, ...string) error:
				rule, err = newRule[int](tt.ruleName, f)
			case func(fmt.Stringer, ...string) error:
				rule, err = newRule[fmt.Stringer](tt.ruleName, f)
			case func(*int, ...string) error:
				rule, err = newRule[*int](tt.ruleName, f)
			case nil:
				// simulate nil fn path
				rule, err = newRule[int](tt.ruleName, nil)
			default:
				panic("unsupported fn type in test")
			}
			tt.assert(rule, err)
		})
	}
}

func TestValidationRuleFn(t *testing.T) {
	// shared errors to assert propagation
	userErr := errors.New("user error")

	// build rules needed for runtime tests
	intRule, _ := newRule[int]("int", func(v int, _ ...string) error { return nil })
	intRuleError, _ := newRule[int]("intErr", func(v int, _ ...string) error { return userErr })
	stringerRule, _ := newRule[fmt.Stringer]("stringer", func(s fmt.Stringer, _ ...string) error { return nil })
	stringerRuleErrMismatch, _ := newRule[fmt.Stringer]("stringerMismatch", func(s fmt.Stringer, _ ...string) error { return nil })
	ifaceRule, _ := newRule[interface{}]("any", func(v interface{}, _ ...string) error { return nil })
	structRule, _ := newRule[structSimple]("struct", func(s structSimple, _ ...string) error { return nil })
	ptrRule, _ := newRule[*int]("ptrInt", func(p *int, _ ...string) error { return nil })

	// Cases for runtime invocation of wrapped fn.
	tests := []struct {
		name             string
		rule             *validationRule
		value            any
		expectedErr      error
		sentinel         error
		propagateUserErr bool
	}{
		{
			name:  "exact match primitive",
			rule:  intRule,
			value: 42,
		},
		{
			name:             "user fn error propagation",
			rule:             intRuleError,
			value:            7,
			expectedErr:      userErr,
			propagateUserErr: true,
		},
		{
			name:        "type mismatch primitive -> mismatch error",
			rule:        intRule,
			value:       "not-int",
			expectedErr: ErrRuleTypeMismatch,
			sentinel:    ErrRuleTypeMismatch,
		},
		{
			name:  "interface assignable concrete type",
			rule:  stringerRule,
			value: myStringerImpl{s: "x"},
		},
		{
			name:        "interface mismatch not implementing",
			rule:        stringerRuleErrMismatch,
			value:       123, // int does not implement fmt.Stringer
			expectedErr: ErrRuleTypeMismatch,
			sentinel:    ErrRuleTypeMismatch,
		},
		{
			name:  "empty interface accepts struct",
			rule:  ifaceRule,
			value: structSimple{A: 1},
		},
		{
			name:        "struct vs pointer mismatch",
			rule:        structRule,
			value:       &structSimple{A: 2},
			expectedErr: ErrRuleTypeMismatch,
			sentinel:    ErrRuleTypeMismatch,
		},
		{
			name:  "pointer exact match",
			rule:  ptrRule,
			value: func() *int { x := 3; return &x }(),
		},
		{
			name:  "nil pointer value accepted",
			rule:  ptrRule,
			value: (*int)(nil),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			fn := tt.rule.getValidationFn()
			err := fn(reflect.ValueOf(tt.value))
			if tt.expectedErr != nil {
				if err == nil {
					if tt.propagateUserErr {
						// impossible: expected user error but got nil
						t.Fatalf("expected user error, got nil")
					}
					// general mismatch expected
					t.Fatalf("expected error %v, got nil", tt.expectedErr)
				}
				if tt.propagateUserErr {
					if !errors.Is(err, tt.expectedErr) {
						if err.Error() != tt.expectedErr.Error() {
							t.Fatalf("expected propagated user error %v, got %v", tt.expectedErr, err)
						}
					}
					return
				}
				// mismatch sentinel path
				if tt.sentinel != nil && !errors.Is(err, tt.sentinel) {
					// final fallback string compare
					if !strings.Contains(err.Error(), tt.sentinel.Error()) {
						t.Fatalf("expected sentinel %v, got %v", tt.sentinel, err)
					}
				}
				return
			}
			if err != nil {
				// unexpected error
				if tt.expectedErr == nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}
