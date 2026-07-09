package core

import (
	"github.com/ygrebnov/model/internal/schema"
	"github.com/ygrebnov/model/validation"
)

func newService[T any]() (*Service[T], error) {
	sc, err := schema.NewController[T]()
	if err != nil {
		return nil, err
	}

	return NewService[T](
		validation.NewRulesRegistry(),
		validation.NewRulesMapping(),
		sc,
		"",
	), nil
}

func newServiceWithEnvPrefix[T any](envPrefix string) (*Service[T], error) {
	sc, err := schema.NewController[T]()
	if err != nil {
		return nil, err
	}

	return NewService[T](
		validation.NewRulesRegistry(),
		validation.NewRulesMapping(),
		sc,
		envPrefix,
	), nil
}
