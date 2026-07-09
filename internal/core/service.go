package core

import (
	"reflect"

	fieldPkg "github.com/ygrebnov/model/field"
	"github.com/ygrebnov/model/internal/schema"
	"github.com/ygrebnov/model/validation"
)

// Service provides per-struct defaulting and validation operations.
type Service[T any] struct {
	rulesRegistry    validation.RulesRegistry
	rulesMapping     validation.RulesMapping
	schemaController schemaController[T]
	envPrefix        string
	envSource        fieldPkg.EnvSource
	// schemas       sync.Map
}

// NewService creates a Service for the given struct type using the
// provided RulesRegistry and RulesMapping instances.
func NewService[T any](
	r validation.RulesRegistry,
	m validation.RulesMapping,
	sc schemaController[T],
	envPrefix string,
) *Service[T] {
	s := &Service[T]{
		rulesRegistry:    r,
		rulesMapping:     m,
		schemaController: sc,
		envPrefix:        envPrefix,
		envSource:        snapshotEnvSource(),
	}

	/*
		if compiled, err := schema.Compile(t); err == nil {
			s.schemas.Store(t, compiled)
		}
	*/

	return s
}

type schemaController[T any] interface {
	GetRoot() *schema.N
	GetFieldType(name string) (reflect.Type, bool)
	GetFieldValue(obj *T, name string) (reflect.Value, bool)
	SetFieldValue(obj *T, name string, value any) bool
}
