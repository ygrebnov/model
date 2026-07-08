package core

import (
	"os"
	"strings"

	fieldPkg "github.com/ygrebnov/model/field"
)

type envSnapshotSource map[string]string

func snapshotEnvSource() fieldPkg.EnvSource {
	env := os.Environ()
	snapshot := make(envSnapshotSource, len(env))
	for _, entry := range env {
		if idx := strings.IndexByte(entry, '='); idx >= 0 {
			snapshot[entry[:idx]] = entry[idx+1:]
			continue
		}
		snapshot[entry] = ""
	}

	return snapshot
}

func (s envSnapshotSource) Lookup(name string) (string, bool) {
	value, ok := s[name]
	return value, ok
}
