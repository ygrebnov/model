package core

import (
	"reflect"
	"sync"

	fieldPkg "github.com/ygrebnov/model/field"
	"github.com/ygrebnov/model/internal/schema"
	"github.com/ygrebnov/model/validation"
)

// Service provides per-struct defaulting and validation operations.
type Service struct {
	// reflectType is the underlying struct type this service was initialized for.
	reflectType   reflect.Type
	rulesRegistry validation.RulesRegistry
	rulesMapping  validation.RulesMapping
	envPrefix     string
	envSource     fieldPkg.EnvSource
	schemas       sync.Map
}

// NewService creates a Service for the given struct type using the
// provided RulesRegistry and RulesMapping instances.
func NewService(
	t reflect.Type,
	r validation.RulesRegistry,
	m validation.RulesMapping,
	envPrefix string,
) *Service {
	s := &Service{
		reflectType:   t,
		rulesRegistry: r,
		rulesMapping:  m,
		envPrefix:     envPrefix,
		envSource:     snapshotEnvSource(),
	}

	if compiled, err := schema.Compile(t); err == nil {
		s.schemas.Store(t, compiled)
	}

	return s
}
