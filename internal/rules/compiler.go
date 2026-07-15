package rules

import (
	"github.com/ygrebnov/model/internal/schema"
)

type Compiler[T any] struct {
	schema   *schema.Schema[T]
	registry *Registry
	rules    map[*schema.Node]nodeValidationRules
}

func NewCompiler[T any](sc *schema.Schema[T], r *Registry) *Compiler[T] {
	return &Compiler[T]{schema: sc, registry: r}
}

type compiledValidationRule struct {
	rule     Rule
	params   []string
	optional bool
}

type Service[T any] struct {
	// ...
	validationRules map[*schema.Node]nodeValidationRules
}

type nodeValidationRules struct {
	field []compiledValidationRule
	elem  []compiledValidationRule
}

func (c *Compiler[T]) Compile(
	root *schema.Node, registry *Registry,
) (map[*schema.Node]nodeValidationRules, error) {
	result := make(map[*schema.Node]nodeValidationRules)

	var walk func(*schema.Node) error
	walk = func(node *schema.Node) error {
		compiled := nodeValidationRules{}

		for _, parsed := range node.ValidateRules {
			rule, err := registry.GetByType(parsed.Name, node.Type)
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
			elemType, hasElements := validationElementType(node.Type)

			if len(node.ValidateElemRules) > 0 && !hasElements {
				return validationElemOnNonCollectionError(node)
			}

			for _, parsed := range node.ValidateElemRules {
				rule, err := registry.GetByType(
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
