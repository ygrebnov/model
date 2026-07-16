package schema

import (
	"reflect"
	"strings"

	"github.com/ygrebnov/errorc"

	"github.com/ygrebnov/model/pkg/errors"
	"github.com/ygrebnov/model/pkg/keys"
)

const (
	tagJSON         = "json"
	tagYAML         = "yaml"
	tagENV          = "env"
	tagValidate     = "validate"
	tagValidateElem = "validateElem"
	tagDefault      = "default"
	tagDefaultElem  = "defaultElem"
)

// Node represents a compiled schema node for one exported struct field.
//
// Name contains the public schema identifier segments. For ordinary nested
// structs, segments are joined with dots, for example "server.host". For
// collection element fields, the collection segment is suffixed with [] to
// show that the child belongs to each element, for example "servers[].host".
//
// Index contains the full reflect.StructField.Index path from the root object
// for ordinary struct fields. For fields below collection elements, Index is
// relative to the collection element type, because runtime traversal must first
// select the concrete slice/array element or map value before using the index.
//
// Reference points to an already compiled node representing the same recursive
// struct type. Recursive nodes do not duplicate the referenced children;
// runtime traversal follows Reference.Children instead.
type Node struct {
	Name                []string
	Type                reflect.Type
	Index               []int
	JSONTag             string
	YAMLTag             string
	Env                 []string
	DefaultTag          string
	DefaultElemTag      string
	ValidateTag         string
	ValidateElemTag     string
	ValidateRules       []Rule
	ValidateElemRules   []Rule
	ValidateElementDive bool

	// Reference points to the already compiled node describing the same
	// struct type when this node closes a recursive type cycle.
	Reference *Node
	Children  []*Node
}

type Rule struct {
	Name     string
	Params   []string
	Optional bool
}

// GetName joins the node name segments with the provided separator and returns
// the public string identifier used by the schema index.
func (n *Node) GetName(separator string) string {
	return strings.Join(n.Name, separator)
}

// Schema owns the compiled schema tree and a string index for field lookup.
//
// The schema is built once by New and then treated as immutable.
// It exposes string-based lookup helpers so callers do not need to know the
// internal reflect.Type, reflect.StructField.Index, or tree representation.
type Schema[T any] struct {
	Tree  *Node
	Index map[string]*Node
}

// addNode registers a compiled node under its public string identifier.
//
// It is used only while the schema is being built, so it does not perform
// locking. After construction the schema is expected to be read-only.
func (s *Schema[T]) addNode(name string, node *Node) {
	s.Index[name] = node
}

// getNode returns the compiled node for a public string identifier.
func (s *Schema[T]) getNode(name string) (*Node, bool) {
	n, ok := s.Index[name]

	return n, ok
}

func (s *Schema[T]) GetRoot() *Node {
	return s.Tree
}

// GetFieldType returns the declared Go type for the field identified by name.
//
// The name must match a compiled schema identifier, such as "server.host" or
// "servers[].host". The boolean result is false when no such field exists.
func (s *Schema[T]) GetFieldType(name string) (reflect.Type, bool) {
	n, ok := s.getNode(name)
	if !ok {
		return nil, false
	}

	return n.Type, true
}

// GetFieldValue resolves a compiled field against a concrete object instance.
//
// It works for ordinary struct fields whose node index is rooted at the object.
// Collection element fields, such as "servers[].host", require runtime
// collection traversal and are therefore not resolved by this helper alone.
func (s *Schema[T]) GetFieldValue(
	obj *T,
	name string,
) (reflect.Value, bool) {
	n, ok := s.getNode(name)
	if !ok {
		return reflect.Value{}, false
	}

	v, ok := valueOf(obj)
	if !ok {
		return reflect.Value{}, false
	}

	v, ok = fieldByIndex(v, n.Index)
	if !ok {
		return reflect.Value{}, false
	}

	return v, true
}

// SetFieldValue sets a concrete object field by public string identifier.
//
// The value is assigned directly when assignable to the field type, converted
// when convertible, or wrapped in a pointer when it matches the pointer
// element type. Passing nil resets the field to its zero value. The boolean
// result reports whether the set was performed.
func (s *Schema[T]) SetFieldValue(
	obj *T,
	name string,
	value any,
) bool {
	fv, ok := s.GetFieldValue(obj, name)
	if !ok || !fv.CanSet() {
		return false
	}

	rv := reflect.ValueOf(value)
	if !rv.IsValid() {
		fv.Set(reflect.Zero(fv.Type()))

		return true
	}

	if rv.Type().AssignableTo(fv.Type()) {
		fv.Set(rv)

		return true
	}

	if rv.Type().ConvertibleTo(fv.Type()) {
		fv.Set(rv.Convert(fv.Type()))

		return true
	}

	if fv.Kind() == reflect.Ptr {
		elemType := fv.Type().Elem()
		if rv.Type().AssignableTo(elemType) {
			pointer := reflect.New(elemType)
			pointer.Elem().Set(rv)
			fv.Set(pointer)

			return true
		}

		if rv.Type().ConvertibleTo(elemType) {
			pointer := reflect.New(elemType)
			pointer.Elem().Set(rv.Convert(elemType))
			fv.Set(pointer)

			return true
		}
	}

	return false
}

// New compiles the schema tree and lookup index for T.
//
// T must be a struct type. Non-struct type parameters return
// errors.ErrTypeParamNotStruct.
func New[T any]() (*Schema[T], error) {
	s := &Schema[T]{
		Index: make(map[string]*Node),
	}

	n, err := newNode(s)
	if err != nil {
		return nil, err
	}

	s.Tree = n

	return s, nil
}

// newNode creates the root schema node and starts parsing the struct type T.
//
// The root node is synthetic: it represents the root object type and does not
// correspond to a concrete struct field.
func newNode[T any](s *Schema[T]) (*Node, error) {
	t := reflect.TypeOf((*T)(nil)).Elem()

	if t.Kind() != reflect.Struct {
		return nil, errors.ErrTypeParamNotStruct
	}

	n := &Node{
		Type: t,
	}

	active := map[reflect.Type]*Node{
		t: n,
	}

	if err := parse(t, n, s, nil, active); err != nil {
		return nil, errorc.With(
			errors.ErrCannotCompileSchema,
			errorc.String(keys.ObjectType, t.String()),
			errorc.Error(keys.Cause, err),
		)
	}

	return n, nil
}

// valueOf returns the concrete struct value behind obj.
//
// The boolean result is false for nil pointers and for non-struct values.
func valueOf[T any](obj *T) (reflect.Value, bool) {
	if obj == nil {
		return reflect.Value{}, false
	}

	v := reflect.ValueOf(obj)

	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return reflect.Value{}, false
		}

		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}

	return v, true
}

// fieldByIndex resolves a possibly nested struct field by an index path.
//
// Pointer-to-struct values along the path are dereferenced. Nil pointers,
// non-struct values, and out-of-range indexes return false.
func fieldByIndex(
	v reflect.Value,
	index []int,
) (reflect.Value, bool) {
	for _, i := range index {
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return reflect.Value{}, false
			}

			v = v.Elem()
		}

		if v.Kind() != reflect.Struct ||
			i < 0 ||
			i >= v.NumField() {
			return reflect.Value{}, false
		}

		v = v.Field(i)
	}

	return v, true
}

// parse walks the exported fields of t and adds their compiled nodes under n.
//
// The parentIndex argument is the index path from the root object to n for
// ordinary struct traversal. For collection element traversal, parentIndex is
// relative to the collection element type, because the concrete element must be
// selected at runtime before the compiled index can be applied.
//
// active maps struct types currently being compiled to their schema nodes. When
// parsing encounters one of those types again, it records a Reference rather
// than recursively duplicating its children.
func parse[T any](
	t reflect.Type,
	n *Node,
	s *Schema[T],
	parentIndex []int,
	active map[reflect.Type]*Node,
) error {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if field.PkgPath != "" {
			continue
		}

		fieldName := strings.ToLower(field.Name)
		index := appendIndex(parentIndex, field.Index)

		validateTag := field.Tag.Get(tagValidate)
		validateElemTag := field.Tag.Get(tagValidateElem)

		validateRules := parseValidateTag(validateTag)
		validateElemRules := parseValidateTag(validateElemTag)

		validateElementDive, validateElemRules :=
			extractValidateElementDive(validateElemRules)

		newN := &Node{
			Name:                appendName(n.Name, fieldName),
			Type:                field.Type,
			Index:               index,
			JSONTag:             field.Tag.Get(tagJSON),
			YAMLTag:             field.Tag.Get(tagYAML),
			Env:                 appendEnv(n.Env, field.Tag.Get(tagENV)),
			DefaultTag:          field.Tag.Get(tagDefault),
			DefaultElemTag:      field.Tag.Get(tagDefaultElem),
			ValidateTag:         validateTag,
			ValidateElemTag:     validateElemTag,
			ValidateRules:       validateRules,
			ValidateElemRules:   validateElemRules,
			ValidateElementDive: validateElementDive,
		}

		n.Children = append(n.Children, newN)

		if childType, ok := nestedStructType(field.Type); ok {
			if referenced, exists := active[childType]; exists {
				newN.Reference = referenced
				s.addNode(newN.GetName("."), newN)

				continue
			}

			s.addNode(newN.GetName("."), newN)
			active[childType] = newN

			if err := parse[T](
				childType,
				newN,
				s,
				index,
				active,
			); err != nil {
				return err
			}

			delete(active, childType)

			continue
		}

		if elemType, ok := collectionElementStructType(
			field.Type,
		); ok {
			newN.Name = collectionName(newN.Name)
			s.addNode(newN.GetName("."), newN)

			if referenced, exists := active[elemType]; exists {
				newN.Reference = referenced

				continue
			}

			active[elemType] = newN

			if err := parse[T](
				elemType,
				newN,
				s,
				nil,
				active,
			); err != nil {
				return err
			}

			delete(active, elemType)

			continue
		}

		s.addNode(newN.GetName("."), newN)
	}

	return nil
}

// appendIndex appends a field index path to a parent index path without
// retaining references to either input slice.
func appendIndex(
	parent []int,
	index []int,
) []int {
	out := make([]int, 0, len(parent)+len(index))
	out = append(out, parent...)
	out = append(out, index...)

	return out
}

// appendName appends a field name segment to a parent schema name path without
// retaining references to the parent slice.
func appendName(
	parent []string,
	name string,
) []string {
	out := make([]string, 0, len(parent)+1)
	out = append(out, parent...)
	out = append(out, name)

	return out
}

// appendEnv appends a raw env tag segment to a parent env path without
// retaining references to the parent slice.
func appendEnv(
	parent []string,
	name string,
) []string {
	out := make([]string, 0, len(parent)+1)
	out = append(out, parent...)
	out = append(out, name)

	return out
}

// nestedStructType returns the struct type represented by t when t is either a
// struct or a pointer to a struct.
func nestedStructType(t reflect.Type) (reflect.Type, bool) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil, false
	}

	return t, true
}

// collectionElementStructType returns the element/value struct type for slices,
// arrays, and maps whose element/value is a struct or pointer to a struct.
func collectionElementStructType(
	t reflect.Type,
) (reflect.Type, bool) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		return nestedStructType(t.Elem())

	case reflect.Map:
		return nestedStructType(t.Elem())

	default:
		return nil, false
	}
}

// collectionName marks the last schema name segment as a collection element
// segment by suffixing it with [].
func collectionName(name []string) []string {
	out := append([]string(nil), name...)
	if len(out) == 0 {
		return out
	}

	out[len(out)-1] += "[]"

	return out
}

// extractValidateElementDive removes the validateElem traversal directive from
// the ordinary element-rule list.
//
// dive is traversal metadata rather than an executable validation rule. It is
// enabled when validateElem contains a parameterless rule named "dive".
func extractValidateElementDive(
	rules []Rule,
) (bool, []Rule) {
	if len(rules) == 0 {
		return false, nil
	}

	filtered := make([]Rule, 0, len(rules))
	dive := false

	for _, rule := range rules {
		if rule.Name == "dive" &&
			len(rule.Params) == 0 {
			dive = true

			continue
		}

		filtered = append(filtered, rule)
	}

	if len(filtered) == 0 {
		return dive, nil
	}

	return dive, filtered
}

// parseValidateTag tokenizes a raw tag string, such as
// "required,min(5),max(10)", into rules.
//
// Behavior:
//   - splits on top-level commas only;
//   - does not split on commas inside parentheses;
//   - trims whitespace around tokens and parameters;
//   - skips empty tokens;
//   - does not support quotes or escaping inside parameters.
func parseValidateTag(tag string) []Rule {
	if tag == "" || tag == "-" {
		return nil
	}

	var tokens []string

	depth := 0
	start := 0

	for i, r := range tag {
		switch r {
		case '(':
			depth++

		case ')':
			if depth > 0 {
				depth--
			}

		case ',':
			if depth == 0 {
				tokens = append(
					tokens,
					strings.TrimSpace(tag[start:i]),
				)
				start = i + 1
			}
		}
	}

	if start <= len(tag) {
		tokens = append(
			tokens,
			strings.TrimSpace(tag[start:]),
		)
	}

	return parseTokens(tokens)
}

func parseTokens(tokens []string) []Rule {
	var rules []Rule

	optional := false

	for _, token := range tokens {
		if token == "" {
			continue
		}

		name := token
		var params []string

		if index := strings.IndexRune(token, '('); index != -1 &&
			strings.HasSuffix(token, ")") {
			name = strings.TrimSpace(token[:index])

			inner := strings.TrimSpace(
				token[index+1 : len(token)-1],
			)

			if inner != "" {
				parts := strings.Split(inner, ",")

				for _, parameter := range parts {
					parameter = strings.TrimSpace(parameter)
					if parameter != "" {
						params = append(params, parameter)
					}
				}
			}
		}

		if name == "omitempty" {
			optional = true

			continue
		}

		if name != "" {
			rules = append(rules, Rule{
				Name:   name,
				Params: params,
			})
		}
	}

	if optional {
		for i := range rules {
			rules[i].Optional = true
		}
	}

	return rules
}
