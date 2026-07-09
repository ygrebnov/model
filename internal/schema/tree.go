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

// N represents a compiled schema node for one exported struct field.
//
// Name contains the public schema identifier segments. For ordinary nested
// structs, segments are joined with dots, for example "server.host". For
// collection element fields, the collection segment is suffixed with [] to
// show that the child belongs to each element, for example "servers[].host".
//
// I contains the full reflect.StructField.Index path from the root object for
// ordinary struct fields. For fields below collection elements, I is relative
// to the collection element type, because runtime traversal must first select
// the concrete slice/array element or map value before using the index.
type N struct {
	Name            []string // all segments (ordered) starting from the root
	T               reflect.Type
	I               []int // reflect.StructField.Index
	JSONTag         string
	YAMLTag         string
	Env             []string // all env tags (ordered) starting from the root
	DefaultTag      string
	DefaultElemTag  string
	ValidateTag     string
	ValidateElemTag string
	Children        []*N
}

// GetName joins the node name segments with the provided separator and returns
// the public string identifier used by the controller index.
func (n *N) GetName(separator string) string {
	return strings.Join(n.Name, separator)
}

// Controller owns the compiled schema tree and a string index for field lookup.
//
// The controller is built once by NewController and then treated as immutable.
// It exposes string-based lookup helpers so callers do not need to know the
// internal reflect.Type, reflect.StructField.Index, or tree representation.
type Controller[T any] struct {
	Tree  *N            // for traversals
	Index map[string]*N // N.fullName (concatenated N.Name) -> *N
}

// addNode registers a compiled node under its public string identifier.
//
// It is used only while the controller is being built, so it does not perform
// locking. After construction the controller is expected to be read-only.
func (c *Controller[T]) addNode(name string, node *N) {
	c.Index[name] = node
}

// getNode returns the compiled node for a public string identifier.
func (c *Controller[T]) getNode(name string) (*N, bool) {
	n, ok := c.Index[name]
	return n, ok
}

func (c *Controller[T]) GetRoot() *N {
	return c.Tree
}

// GetFieldType returns the declared Go type for the field identified by name.
//
// The name must match a compiled schema identifier, such as "server.host" or
// "servers[].host". The boolean result is false when no such field exists.
func (c *Controller[T]) GetFieldType(name string) (reflect.Type, bool) {
	n, ok := c.getNode(name)
	if !ok {
		return nil, false
	}

	return n.T, true
}

// GetFieldValue resolves a compiled field against a concrete object instance.
//
// It works for ordinary struct fields whose node index is rooted at the object.
// Collection element fields, such as "servers[].host", require runtime
// collection traversal and are therefore not resolved by this helper alone.
func (c *Controller[T]) GetFieldValue(obj *T, name string) (reflect.Value, bool) {
	n, ok := c.getNode(name)
	if !ok {
		return reflect.Value{}, false
	}

	v, ok := valueOf(obj)
	if !ok {
		return reflect.Value{}, false
	}

	v, ok = fieldByIndex(v, n.I)
	if !ok {
		return reflect.Value{}, false
	}

	return v, true
}

// SetFieldValue sets a concrete object field by public string identifier.
//
// The value is assigned directly when assignable to the field type, converted
// when convertible, or rejected when neither is possible. Passing nil resets
// the field to its zero value. The boolean result reports whether the set was
// performed.
func (c *Controller[T]) SetFieldValue(obj *T, name string, value any) bool {
	fv, ok := c.GetFieldValue(obj, name)
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

	return false
}

// NewController compiles the schema tree and lookup index for T.
//
// T may be a struct type or a pointer to a struct type. Non-struct type
// parameters return errors.ErrTypeParamNotStruct.
func NewController[T any]() (*Controller[T], error) {
	c := &Controller[T]{
		Index: make(map[string]*N),
	}

	n, err := newNode(c)
	if err != nil {
		return nil, err
	}
	c.Tree = n
	return c, nil
}

// newNode creates the root schema node and starts parsing the struct type T.
//
// The root node is synthetic: it represents the root object type and does not
// correspond to a concrete struct field.
func newNode[T any](c *Controller[T]) (*N, error) {
	t := reflect.TypeOf((*T)(nil)).Elem()
	t = dereferenceType(t)

	if t.Kind() != reflect.Struct {
		return nil, errors.ErrTypeParamNotStruct
	}

	n := &N{
		T: t,
	}

	active := map[reflect.Type]bool{t: true}
	if err := parse(t, n, c, nil, active); err != nil {
		return nil, errorc.With(
			errors.ErrCannotCompileSchema,
			errorc.String(keys.ObjectType, t.String()),
			errorc.Error(keys.Cause, err),
		)
	}

	return n, nil
}

// dereferenceType unwraps pointer types until it reaches a non-pointer type.
func dereferenceType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
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
func fieldByIndex(v reflect.Value, index []int) (reflect.Value, bool) {
	for _, i := range index {
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return reflect.Value{}, false
			}
			v = v.Elem()
		}

		if v.Kind() != reflect.Struct || i < 0 || i >= v.NumField() {
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
// The active map prevents infinite recursion for self-referential types.
func parse[T any](t reflect.Type, n *N, c *Controller[T], parentIndex []int, active map[reflect.Type]bool) error {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// Skip unexported fields.
		if field.PkgPath != "" {
			continue
		}

		fieldName := strings.ToLower(field.Name)
		index := appendIndex(parentIndex, field.Index)

		newN := &N{
			Name:            appendName(n.Name, fieldName),
			T:               field.Type,
			I:               index,
			JSONTag:         field.Tag.Get(tagJSON),
			YAMLTag:         field.Tag.Get(tagYAML),
			Env:             appendEnv(n.Env, field.Tag.Get(tagENV)),
			DefaultTag:      field.Tag.Get(tagDefault),
			DefaultElemTag:  field.Tag.Get(tagDefaultElem),
			ValidateTag:     field.Tag.Get(tagValidate),
			ValidateElemTag: field.Tag.Get(tagValidateElem),
		}

		n.Children = append(n.Children, newN)
		c.addNode(newN.GetName("."), newN)

		if childType, ok := nestedStructType(field.Type); ok {
			if active[childType] {
				continue
			}

			active[childType] = true
			if err := parse[T](childType, newN, c, index, active); err != nil {
				return err
			}
			delete(active, childType)
			continue
		}

		if elemType, ok := collectionElementStructType(field.Type); ok {
			if active[elemType] {
				continue
			}

			collectionNode := newN
			collectionNode.Name = collectionName(collectionNode.Name)

			active[elemType] = true
			if err := parse[T](elemType, collectionNode, c, nil, active); err != nil {
				return err
			}
			delete(active, elemType)
			continue
		}

		if field.Type.Kind() == reflect.Interface {
			continue
		}
	}
	return nil
}

// appendIndex appends a field index path to a parent index path without
// retaining references to either input slice.
func appendIndex(parent []int, index []int) []int {
	out := make([]int, 0, len(parent)+len(index))
	out = append(out, parent...)
	out = append(out, index...)
	return out
}

// appendName appends a field name segment to a parent schema name path without
// retaining references to the parent slice.
func appendName(parent []string, name string) []string {
	out := make([]string, 0, len(parent)+1)
	out = append(out, parent...)
	out = append(out, name)
	return out
}

// appendEnv appends a raw env tag segment to a parent env path without
// retaining references to the parent slice.
func appendEnv(parent []string, name string) []string {
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

/*
func nestedStructType(t reflect.Type) (reflect.Type, bool) {
	switch t.Kind() {
	case reflect.Struct:
		return t, true
	case reflect.Ptr:
		if elem := t.Elem(); elem.Kind() == reflect.Struct {
			return elem, true
		}
	}

	return nil, false
}
*/

// collectionElementStructType returns the element/value struct type for slices,
// arrays, and maps whose element/value is a struct or pointer to a struct.
func collectionElementStructType(t reflect.Type) (reflect.Type, bool) {
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

	out[len(out)-1] = out[len(out)-1] + "[]"
	return out
}
