package schema

import (
	"reflect"
	"strings"

	modelerrors "github.com/ygrebnov/model/pkg/errors"
)

// Schema is a compiled field tree for a struct type.
type Schema struct {
	Type  reflect.Type
	Root  *Node
	nodes []*Node
	index map[string]*Node
}

// Node describes one struct field in the compiled schema tree.
type Node struct {
	Name            string
	Path            string
	Type            reflect.Type
	Index           []int
	Depth           int
	Parent          *Node
	Children        []*Node
	Anonymous       bool
	Recursive       bool
	JSONName        string
	EnvName         string
	EnvEnabled      bool
	EnvPath         []string
	DefaultTag      string
	DefaultElemTag  string
	ValidateTag     string
	ValidateElemTag string

	childrenByName map[string]*Node
}

// Compile builds a finite field tree for the provided struct type.
//
// Pointer roots are unwrapped until a non-pointer type is reached. Recursive
// edges are preserved as nodes but not expanded further; such nodes are marked
// with Recursive=true so the compiled graph remains a tree.
func Compile(t reflect.Type) (*Schema, error) {
	t = dereferenceType(t)
	if t == nil || t.Kind() != reflect.Struct {
		return nil, modelerrors.ErrTypeParamNotStruct
	}

	root := &Node{
		Name:           t.Name(),
		Type:           t,
		childrenByName: make(map[string]*Node),
	}

	s := &Schema{
		Type:  t,
		Root:  root,
		index: make(map[string]*Node),
	}

	state := compileState{
		schema: s,
		active: map[reflect.Type]int{t: 1},
	}
	state.buildChildren(root, t, nil, true)

	return s, nil
}

// Lookup returns the node for a dotted Go field path.
func (s *Schema) Lookup(path string) (*Node, bool) {
	if s == nil || s.Root == nil {
		return nil, false
	}
	if path == "" {
		return s.Root, true
	}

	node, ok := s.index[path]
	return node, ok
}

// Fields returns all compiled field nodes in preorder.
func (s *Schema) Fields() []*Node {
	if s == nil || len(s.nodes) == 0 {
		return nil
	}

	out := make([]*Node, len(s.nodes))
	copy(out, s.nodes)
	return out
}

// Child returns the direct child with the given Go field name.
func (n *Node) Child(name string) (*Node, bool) {
	if n == nil {
		return nil, false
	}

	child, ok := n.childrenByName[name]
	return child, ok
}

// StructType reports the nested struct type represented by the node when the
// current traversal logic would recurse into it.
func (n *Node) StructType() (reflect.Type, bool) {
	if n == nil {
		return nil, false
	}

	return nestedStructType(n.Type)
}

// IsCollection reports whether the node is a slice, array, or map, optionally
// wrapped by a single pointer.
func (n *Node) IsCollection() bool {
	if n == nil {
		return false
	}

	t := n.Type
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		return true
	default:
		return false
	}
}

// CollectionElementType reports the element type for slice/array/map nodes,
// optionally wrapped by a single pointer.
func (n *Node) CollectionElementType() (reflect.Type, bool) {
	if n == nil {
		return nil, false
	}

	t := n.Type
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		return t.Elem(), true
	default:
		return nil, false
	}
}

// MapKeyType reports the map key type for map nodes, optionally wrapped by a
// single pointer.
func (n *Node) MapKeyType() (reflect.Type, bool) {
	if n == nil {
		return nil, false
	}

	t := n.Type
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Map {
		return nil, false
	}

	return t.Key(), true
}

type compileState struct {
	schema *Schema
	active map[reflect.Type]int
}

func (s *compileState) buildChildren(parent *Node, typ reflect.Type, envPath []string, envEnabled bool) {
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue
		}

		path := field.Name
		if parent.Path != "" {
			path = parent.Path + "." + field.Name
		}

		envName, useEnv := effectiveEnvName(field)
		childEnvEnabled := envEnabled && useEnv
		var childEnvPath []string
		if childEnvEnabled {
			childEnvPath = appendEnvPath(envPath, envName)
		}

		node := &Node{
			Name:            field.Name,
			Path:            path,
			Type:            field.Type,
			Index:           append([]int(nil), field.Index...),
			Depth:           parent.Depth + 1,
			Parent:          parent,
			Anonymous:       field.Anonymous,
			JSONName:        jsonTagName(field),
			EnvName:         envName,
			EnvEnabled:      childEnvEnabled,
			EnvPath:         append([]string(nil), childEnvPath...),
			DefaultTag:      normalizedTag(field.Tag.Get(tagDefault)),
			DefaultElemTag:  normalizedTag(field.Tag.Get(tagDefaultElem)),
			ValidateTag:     normalizedTag(field.Tag.Get(tagValidate)),
			ValidateElemTag: normalizedTag(field.Tag.Get(tagValidateElem)),
			childrenByName:  make(map[string]*Node),
		}

		parent.Children = append(parent.Children, node)
		parent.childrenByName[node.Name] = node
		s.schema.nodes = append(s.schema.nodes, node)
		s.schema.index[node.Path] = node

		childType, ok := nestedStructType(field.Type)
		if !ok {
			continue
		}
		if s.active[childType] > 0 {
			node.Recursive = true
			continue
		}

		s.active[childType]++
		s.buildChildren(node, childType, childEnvPath, childEnvEnabled)
		s.active[childType]--
	}
}

func dereferenceType(t reflect.Type) reflect.Type {
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

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

func effectiveEnvName(field reflect.StructField) (string, bool) {
	if tag := field.Tag.Get("env"); tag != "" {
		if tag == "-" {
			return "", false
		}
		return strings.ToUpper(tag), true
	}

	if name := jsonTagName(field); name != "" {
		return strings.ToUpper(name), true
	}

	return strings.ToUpper(field.Name), true
}

func jsonTagName(field reflect.StructField) string {
	name := tagName(field.Tag.Get("json"))
	if name == "-" {
		return ""
	}

	return name
}

func tagName(tag string) string {
	if idx := strings.IndexByte(tag, ','); idx >= 0 {
		return tag[:idx]
	}

	return tag
}

func normalizedTag(tag string) string {
	if tag == "" || tag == "-" {
		return ""
	}

	return tag
}

func appendEnvPath(parent []string, part string) []string {
	path := make([]string, 0, len(parent)+1)
	path = append(path, parent...)
	path = append(path, part)
	return path
}
