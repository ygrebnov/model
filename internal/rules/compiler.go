package rules

import "github.com/ygrebnov/model/internal/schema"

type Compiler[T any] struct {
	schema   *schema.Controller[T]
	registry *Registry
	rules    map[*schema.N]nodeValidationRules
}

func NewCompiler[T any](sc *schema.Controller[T], r *Registry) *Compiler[T] {
	return &Compiler[T]{schema: sc, registry: r}
}
