package model

import (
	"context"
	"reflect"
)

// ensureBinding initializes the model's typeBinding, rulesRegistry, and rulesMapping lazily.
func (m *Model[TObject]) ensureBinding() error {
	if m.binding != nil {
		return nil
	}
	// Derive the concrete struct type from the bound object.
	rv, err := m.rootStructValue("initBinding")
	if err != nil {
		return err
	}
	typ := rv.Type()
	reg := newRulesRegistry()
	mapping := newRulesMapping()
	tb, err := buildTypeBinding(typ, reg, mapping)
	if err != nil {
		return err
	}
	m.binding = tb
	return nil
}

// RegisterRules registers one or many named custom validation rules of the same field type
// into the model's validator rulesRegistry.
//
// See the Rule type and NewRule function for details on creating rules.
func (m *Model[TObject]) RegisterRules(rules ...Rule) error {
	if err := m.ensureBinding(); err != nil {
		return err
	}
	for _, rule := range rules {
		if err := m.binding.rulesRegistry.add(rule); err != nil {
			return err
		}
	}
	return nil
}

// Validate runs the registered validation rules against the model's bound object with the provided context.
// If the context is canceled or its deadline exceeded, validation stops early and ctx.Err() is returned.
func (m *Model[TObject]) Validate(ctx context.Context) error {
	if err := m.ensureBinding(); err != nil {
		return err
	}
	return m.validate(ctx)
}

// validate is the internal implementation that walks struct fields and applies rules
// declared in `validate:"..."` tags. It supports rule parameters via the syntax
// "rule" or "rule(p1,p2)" and multiple rules separated by commas.
func (m *Model[TObject]) validate(ctx context.Context) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := m.ensureBinding(); err != nil {
		return err
	}

	var rv reflect.Value
	if rv, err = m.rootStructValue("Validate"); err != nil {
		return err
	}
	ve := &ValidationError{}
	if err := m.binding.validateStruct(ctx, rv, "", ve); err != nil {
		return err
	}
	if ve.Empty() {
		return nil
	}
	return ve
}

// The following methods remain for now to keep tests and behavior stable.
// They are no longer used by the core validation path after delegation to
// typeBinding, but may still be referenced in tests. They can be removed or
// redirected to typeBinding in a follow-up cleanup if desired.

// applyRule fetches the named rule from the registry and applies it to the given reflect.Value v,
// passing any additional string parameters.
// If the rule is not found or fails, an error is returned.
func (m *Model[TObject]) applyRule(name string, v reflect.Value, params ...string) error {
	r, err := m.binding.rulesRegistry.get(name, v)
	if err != nil {
		return err
	}

	return r.getValidationFn()(v, params...)
}

// validateStruct walks a struct value and applies rules on each field according to its `validate` tag.
// Nested structs and pointers to structs are traversed recursively. The `path` argument tracks the
// dotted field path for clearer error messages.
func (m *Model[TObject]) validateStruct(ctx context.Context, rv reflect.Value, path string, ve *ValidationError) error {
	// Delegate to typeBinding for actual traversal to keep behavior centralized.
	return m.binding.validateStruct(ctx, rv, path, ve)
}

// validateElements applies validation rules to elements of a slice, array, or map
// using pre-parsed rules (e.g., retrieved from the cache).
func (m *Model[TObject]) validateElements(ctx context.Context, fv reflect.Value, fpath string, rules []ruleNameParams, ve *ValidationError) error {
	return m.binding.validateElements(ctx, fv, fpath, rules, ve)
}

// validateSingleElement handles validation for a single item from a collection.
func (m *Model[TObject]) validateSingleElement(ctx context.Context, elem reflect.Value, path string, rules []ruleNameParams, isDiveOnly bool, ve *ValidationError) error {
	return m.binding.validateSingleElement(ctx, elem, path, rules, isDiveOnly, ve)
}
