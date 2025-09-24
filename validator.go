package model

import (
	"fmt"
	"reflect"

	"github.com/ygrebnov/model/rule"
)

// rulesRegistry is a per-model registry of validation rules.
type rulesRegistry struct {
	rules map[string][]ruleAdapter // per-model registry: rule name -> overloads by type
}

// addRule adds a validation rule to the validator.
func (v *validator) addRule(rule rule.Rule[TField]) error {
	return nil
}

// validate applies all registered validation rules to the given object.
func (v *validator) validate(obj interface{}) error {
	return nil
}
