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

// applyRule fetches the named rule from the registry and applies it to the given reflect.Value v,
// passing any additional string parameters.
// If the rule is not found or fails, an error is returned.
func (tb *typeBinding) applyRule(name string, v reflect.Value, params ...string) error {
	r, err := tb.rulesRegistry.get(name, v)
	if err != nil {
		return err
	}
	return r.getValidationFn()(v, params...)
}
