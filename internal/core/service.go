package core

import (
	"reflect"

	"github.com/ygrebnov/model/validation"
)

// Service provides per-struct defaulting and validation operations.
type Service struct {
	// reflectType is the underlying struct type this service was initialized for.
	reflectType   reflect.Type
	rulesRegistry validation.RulesRegistry
	rulesMapping  validation.RulesMapping
	envPrefix     string
	envValues     map[string]string
}

// NewService creates a Service for the given struct type using the
// provided RulesRegistry and RulesMapping instances.
func NewService(
	t reflect.Type,
	r validation.RulesRegistry,
	m validation.RulesMapping,
	envPrefix string,
) *Service {
	return &Service{
		reflectType:   t,
		rulesRegistry: r,
		rulesMapping:  m,
		envPrefix:     envPrefix,
		envValues:     snapshotEnv(),
	}
}
