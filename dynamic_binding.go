package model

import (
	"context"
	"reflect"

	"github.com/ygrebnov/errorc"

	"github.com/ygrebnov/model/field"
	"github.com/ygrebnov/model/pkg/errors"
	"github.com/ygrebnov/model/pkg/keys"
	"github.com/ygrebnov/model/validation"
)

// DynamicBinding is a Binding invariant with type known only on runtime.
type DynamicBinding struct {
	t       reflect.Type
	service service
}

// NewDynamicBinding constructs a DynamicBinding instance resolving type from the given object pointer.
//
// Use WithEnvPrefix option to add an environment variable name prefix for ApplyEnv
// and ValidateWithDefaults.
//
// Builtin rules are applied implicitly.
//
// Use WithRules option to register custom validation rules at construction time.
// See [validation.Rule] and [validation.NewRule] for details on creating rules.
func NewDynamicBinding(obj any, opts ...Option) (*DynamicBinding, error) {
	if obj == nil {
		return nil, errors.ErrNilObject
	}
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return nil, errors.ErrNotStructPtr
	}
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return nil, errors.ErrNotStructPtr
	}

	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	rulesRegistry := validation.NewRulesRegistry()
	rulesMapping := validation.NewRulesMapping()

	s := newService(elem.Type(), rulesRegistry, rulesMapping, o.envPrefix)

	b := &DynamicBinding{t: v.Type(), service: s}
	if len(o.rules) > 0 {
		if err := registerRules(s, o.rules...); err != nil {
			return nil, err
		}
	}

	return b, nil
}

func (b *DynamicBinding) validateTarget(obj any) (reflect.Value, error) {
	if obj == nil {
		return reflect.Value{}, errors.ErrNilObject
	}
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return reflect.Value{}, errors.ErrNotStructPtr
	}
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return reflect.Value{}, errors.ErrNotStructPtr
	}

	if v.Type() != b.t {
		return reflect.Value{}, errorc.With(
			errors.ErrTypeMismatch,
			errorc.String(keys.ObjectType, v.Type().String()),
			errorc.String(keys.ExpectedType, b.t.String()),
		)
	}
	return elem, nil
}

// ApplyDefaults applies default values to zero fields of obj according to its `default` / `defaultElem` tags.
// ApplyDefaults applies defaults each time it is called.
// It is idempotent, but not once-guarded.
func (b *DynamicBinding) ApplyDefaults(obj any) error {
	v, err := b.validateTarget(obj)
	if err != nil {
		return err
	}

	return b.service.SetDefaultsStruct(v)
}

// ApplyEnv applies environment-backed values from source to obj using compiled field metadata.
func (b *DynamicBinding) ApplyEnv(obj any, source field.EnvSource) error {
	v, err := b.validateTarget(obj)
	if err != nil {
		return err
	}

	return b.service.ApplyEnvStruct(v, source)
}

// Validate runs validation rules declared via `validate` / `validateElem` tags on obj
// with the provided context.
//
// If validation fails, a *validation.Error is returned; if the context is canceled, ctx.Err() is returned.
func (b *DynamicBinding) Validate(ctx context.Context, obj any) error {
	v, err := b.validateTarget(obj)
	if err != nil {
		return err
	}

	if ctx == nil {
		ctx = context.Background()
	}

	ve := &validation.Error{}
	if err := b.service.ValidateStruct(ctx, v, "", ve); err != nil {
		return err
	}
	if ve.Empty() {
		return nil
	}
	return ve
}

// ValidateWithDefaults first applies defaults and snapshotted environment-backed values to obj,
// then runs validation. This is a convenience for service-level flows that expect resolved inputs
// before validation.
func (b *DynamicBinding) ValidateWithDefaults(ctx context.Context, obj any) error {
	v, err := b.validateTarget(obj)
	if err != nil {
		return err
	}

	if err := b.service.SetDefaultsStruct(v); err != nil {
		return err
	}
	if err := b.service.ApplySnapshotEnvStruct(v); err != nil {
		return err
	}
	return b.Validate(ctx, obj)
}
