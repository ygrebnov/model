package core

import (
	"os"
	"reflect"

	fieldPkg "github.com/ygrebnov/model/field"
)

// snapshotEnvSource creates an immutable snapshot containing only environment
// variables represented by scalar fields in the compiled schema.
func (s *Service[T]) snapshotEnvSource() fieldPkg.EnvSource {
	snapshot := make(envSnapshotSource)

	root := reflect.New(reflect.TypeFor[T]()).Elem()
	policy := walkPolicy{
		DiveCollection: func(walkContext, reflect.Value) bool {
			return false
		},
		AllocPtrStruct: func(ctx walkContext, _ reflect.Value) bool {
			return ctx.EnvEnabled
		},
	}

	_ = walkSchema(
		root,
		s.schema.GetRoot(),
		envPrefixPath(s.envPrefix),
		policy,
		func(ctx walkContext, field reflect.Value) error {
			if !ctx.EnvEnabled || !canSetLiteralValue(field) {
				return nil
			}

			envName := joinEnvPath(ctx.EnvPath)
			if envName == "" {
				return nil
			}

			if value, ok := os.LookupEnv(envName); ok {
				snapshot[envName] = value
			}

			return nil
		},
	)

	return snapshot
}

type envSnapshotSource map[string]string

func (s envSnapshotSource) Lookup(name string) (string, bool) {
	value, ok := s[name]
	return value, ok
}
