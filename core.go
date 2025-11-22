package model

import "reflect"

type typeBinding struct {
	// typ is the underlying struct type this binding was built for.
	typ           reflect.Type
	rulesRegistry rulesRegistry
	rulesMapping  rulesMapping
}

// buildTypeBinding creates a typeBinding for the given struct type using the
// provided registry and mapping instances. For now it wires the provided
// registry/mapping into the binding; any future per-type precomputation can be
// added here.
func buildTypeBinding(typ reflect.Type, reg rulesRegistry, mapping rulesMapping) (*typeBinding, error) {
	return &typeBinding{
		typ:           typ,
		rulesRegistry: reg,
		rulesMapping:  mapping,
	}, nil
}
