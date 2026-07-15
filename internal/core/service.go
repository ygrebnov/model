package core

import (
	"reflect"

	"github.com/ygrebnov/errorc"

	fieldPkg "github.com/ygrebnov/model/field"
	"github.com/ygrebnov/model/internal/rules"
	"github.com/ygrebnov/model/internal/schema"
	"github.com/ygrebnov/model/pkg/errors"
	"github.com/ygrebnov/model/pkg/keys"
)

// Service provides per-struct defaulting and validation operations.
type Service[T any] struct {
	schema     schemaService[T]
	validation validationPlan
	envPrefix  string
	envSource  fieldPkg.EnvSource
}

// NewService creates a fully initialized Service for the given struct type.
//
// Validation rules are resolved against the compiled schema first. Environment
// values are snapshotted only after validation-plan compilation succeeds.
func NewService[T any](
	registry *rules.Registry,
	ss schemaService[T],
	envPrefix string,
) (*Service[T], error) {
	validation, err := newValidationPlan(ss.GetRoot(), registry)
	if err != nil {
		return nil, err
	}

	s := &Service[T]{
		schema:     ss,
		validation: validation,
		envPrefix:  envPrefix,
	}

	s.envSource = s.snapshotEnvSource()

	return s, nil
}

type schemaService[T any] interface {
	GetRoot() *schema.Node
	GetFieldType(name string) (reflect.Type, bool)
	GetFieldValue(obj *T, name string) (reflect.Value, bool)
	SetFieldValue(obj *T, name string, value any) bool
}

type validationPlan struct {
	nodes map[*schema.Node]compiledNodeRules
}

type compiledNodeRules struct {
	field []compiledRule
	elem  []compiledRule
	dive  bool
}

type compiledRule struct {
	rule     *rules.Rule
	params   []string
	optional bool
}

// newValidationPlan resolves all parsed validation metadata in the schema into
// executable rules. It fails during Service construction when a rule is
// missing, has no overload for the declared field type, or validateElem is used
// on a non-collection field.
func newValidationPlan(
	root *schema.Node,
	registry *rules.Registry,
) (validationPlan, error) {
	result := validationPlan{
		nodes: make(map[*schema.Node]compiledNodeRules),
	}

	if root == nil {
		return result, nil
	}

	var compileNode func(*schema.Node) error
	compileNode = func(node *schema.Node) error {
		if node == nil {
			return nil
		}

		compiled, err := compileNodeValidationRules(node, registry)
		if err != nil {
			return err
		}

		result.nodes[node] = compiled

		for _, child := range node.Children {
			if err := compileNode(child); err != nil {
				return err
			}
		}

		return nil
	}

	if err := compileNode(root); err != nil {
		return validationPlan{}, err
	}

	return result, nil
}

func compileNodeValidationRules(
	node *schema.Node,
	registry *rules.Registry,
) (compiledNodeRules, error) {
	compiled := compiledNodeRules{
		dive: node.ValidateElemDive,
	}

	for _, parsed := range node.ValidateRules {
		rule, err := registry.GetByType(parsed.Name, node.Type)
		if err != nil {
			return compiledNodeRules{}, err
		}

		compiled.field = append(compiled.field, compiledRule{
			rule:     rule,
			params:   append([]string(nil), parsed.Params...),
			optional: parsed.Optional,
		})
	}

	if node.ValidateElemDive {
		return compiled, nil
	}

	if len(node.ValidateElemRules) == 0 {
		return compiled, nil
	}

	elemType, ok := validationElementType(node.Type)
	if !ok {
		return compiledNodeRules{}, validationElemOnNonCollectionError(node)
	}

	for _, parsed := range node.ValidateElemRules {
		rule, err := registry.GetByType(parsed.Name, elemType)
		if err != nil {
			return compiledNodeRules{}, err
		}

		compiled.elem = append(compiled.elem, compiledRule{
			rule:     rule,
			params:   append([]string(nil), parsed.Params...),
			optional: parsed.Optional,
		})
	}

	return compiled, nil
}

// validationElementType returns the element type validated by validateElem.
// Pointer-to-collection fields are dereferenced, while pointer element types are
// preserved because rule overload resolution must match the actual element
// value passed during validation.
func validationElementType(t reflect.Type) (reflect.Type, bool) {
	if t == nil {
		return nil, false
	}

	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		return t.Elem(), true
	default:
		return nil, false
	}
}

func validationElemOnNonCollectionError(node *schema.Node) error {
	if node == nil {
		return errors.ErrInvalidValidateElemUsage
	}

	return errorc.With(
		errors.ErrInvalidValidateElemUsage,
		errorc.String(keys.FieldName, node.GetName(".")),
		errorc.String(keys.FieldType, node.Type.String()),
	)
}
