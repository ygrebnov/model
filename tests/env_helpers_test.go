package tests

import (
	"os"

	"github.com/ygrebnov/model"
)

type osEnvSource struct{}

func (osEnvSource) Lookup(name string) (string, bool) {
	return os.LookupEnv(name)
}

func applyBindingDefaultsAndEnv[T any](b *model.Binding[T], obj *T) error {
	if err := b.ApplyDefaults(obj); err != nil {
		return err
	}

	return b.ApplyEnv(obj, osEnvSource{})
}

func applyDynamicDefaultsAndEnv(b *model.DynamicBinding, obj any) error {
	if err := b.ApplyDefaults(obj); err != nil {
		return err
	}

	return b.ApplyEnv(obj, osEnvSource{})
}
