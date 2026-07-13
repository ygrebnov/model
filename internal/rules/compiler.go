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

func (c *Compiler[T]) Compile(
	root *schema.N, registry *Registry,
) (map[*schema.N]nodeValidationRules, error) {
	result := make(map[*schema.N]nodeValidationRules)

	var walk func(*schema.N) error
	walk = func(node *schema.N) error {
		compiled := nodeValidationRules{}

		for _, parsed := range node.ValidateRules {
			rule, err := registry.Resolve(parsed.Name, node.T)
			if err != nil {
				return nilRuleCompileError(node, parsed, err)
			}

			compiled.field = append(
				compiled.field,
				compiledValidationRule{
					rule:     rule,
					params:   parsed.Params,
					optional: parsed.Optional,
				},
			)
		}

		if !node.ValidateElemDive {
			elemType, hasElements := validationElementType(node.T)

			if len(node.ValidateElemRules) > 0 && !hasElements {
				return validationElemOnNonCollectionError(node)
			}

			for _, parsed := range node.ValidateElemRules {
				rule, err := registry.Resolve(
					parsed.Name,
					elemType,
				)
				if err != nil {
					return nilRuleCompileError(
						node,
						parsed,
						err,
					)
				}

				compiled.elem = append(
					compiled.elem,
					compiledValidationRule{
						rule:     rule,
						params:   parsed.Params,
						optional: parsed.Optional,
					},
				)
			}
		}

		result[node] = compiled

		for _, child := range node.Children {
			if err := walk(child); err != nil {
				return err
			}
		}

		return nil
	}

	if err := walk(root); err != nil {
		return nil, err
	}

	return result, nil

}
