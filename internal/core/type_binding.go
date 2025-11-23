package core

import (
	"reflect"

	"github.com/ygrebnov/model/validation"
)

type TypeBinding struct {
	// reflectType is the underlying struct type this binding was built for.
	reflectType   reflect.Type
	rulesRegistry validation.Registry
	rulesMapping  validation.Mapping
}

// NewTypeBinding creates a TypeBinding for the given struct type using the
// provided registry and mapping instances. For now, it wires the provided
// registry/mapping into the binding; any future per-type precomputation can be
// added here.
func NewTypeBinding(t reflect.Type, r validation.Registry, m validation.Mapping) (*TypeBinding, error) {
	return &TypeBinding{
		reflectType:   t,
		rulesRegistry: r,
		rulesMapping:  m,
	}, nil
}
