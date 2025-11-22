package core

import (
	"reflect"

	"github.com/ygrebnov/model/internal/rules"
)

type TypeBinding struct {
	// typ is the underlying struct type this binding was built for.
	typ           reflect.Type
	rulesRegistry rulesRegistry
	rulesMapping  rulesMapping
}

// NewTypeBinding creates a TypeBinding for the given struct type using the
// provided registry and mapping instances. For now it wires the provided
// registry/mapping into the binding; any future per-type precomputation can be
// added here.
func NewTypeBinding(typ reflect.Type, reg RulesRegistry, mapping RulesMapping) (*TypeBinding, error) {
	return &TypeBinding{
		typ:           typ,
		rulesRegistry: reg,
		rulesMapping:  mapping,
	}, nil
}

type RulesRegistry interface {
	Add(r rules.Rule) error
	Get(name string, v reflect.Value) (rules.Rule, error)
}

func NewRulesRegistry() RulesRegistry {
	return rules.NewRegistry()
}

type RulesMapping interface {
	add(parent reflect.Type, fieldIndex int, tagName string, rules []ruleNameParams)
	get(parent reflect.Type, fieldIndex int, tagName string) ([]ruleNameParams, bool)
}

func newRulesMapping() rulesMapping {
	return newMapping()
}
