package core

import (
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

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
			// Recursive references have no finite environment path to snapshot.
			// Avoid allocating a new pointer at the cycle boundary.
			return ctx.EnvEnabled && ctx.Node.Reference == nil
		},
	}

	_ = walkSchema(
		root,
		s.schema.GetRoot(),
		envPrefixPath(s.envPrefix),
		policy,
		func(ctx walkContext, field reflect.Value) error {
			if !ctx.EnvEnabled {
				return nil
			}

			envName := joinEnvPath(ctx.EnvPath)
			if envName == "" {
				return nil
			}

			if isCollectionNode(ctx.Node) {
				snapshotEnvironmentPrefix(snapshot, envName)

				return nil
			}

			if !canSetLiteralValue(field) {
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

func snapshotEnvironmentPrefix(snapshot envSnapshotSource, prefix string) {
	prefix += "_"

	for _, entry := range os.Environ() {
		name, value, ok := strings.Cut(entry, "=")
		if ok && strings.HasPrefix(name, prefix) {
			snapshot[name] = value
		}
	}
}

type envSnapshotSource map[string]string

func (s envSnapshotSource) Lookup(name string) (string, bool) {
	value, ok := s[name]
	return value, ok
}

// CollectionIndexes returns the indexed descendants captured below prefix.
func (s envSnapshotSource) CollectionIndexes(prefix string) []int {
	prefix += "_"
	indexSet := make(map[int]struct{})

	for name := range s {
		if !strings.HasPrefix(name, prefix) {
			continue
		}

		indexPart, _, ok := strings.Cut(
			strings.TrimPrefix(name, prefix),
			"_",
		)
		if !ok {
			continue
		}

		index, err := strconv.Atoi(indexPart)
		if err != nil || index < 0 || strconv.Itoa(index) != indexPart {
			continue
		}

		indexSet[index] = struct{}{}
	}

	indexes := make([]int, 0, len(indexSet))
	for index := range indexSet {
		indexes = append(indexes, index)
	}

	sort.Ints(indexes)

	return indexes
}
