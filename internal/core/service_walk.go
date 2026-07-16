package core

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/ygrebnov/model/internal/schema"
)

// walkContext describes one concrete visit produced by walkSchema.
//
// Node is the compiled schema node. Path is the runtime field path, for example
// "server.host", "servers[0].host", or "servers[api].host". EnvPath is the
// runtime environment path for the same field.
//
// EnvEnabled indicates whether environment lookup is enabled for the node.
// An explicit env:"-" disables environment lookup for that node and all of its
// descendants.
type walkContext struct {
	Node       *schema.Node
	Path       string
	EnvPath    []string
	EnvEnabled bool
}

// walkPolicy contains operation-specific traversal decisions.
//
// Defaults, env application, validation, and provider-based value application
// use the same reflection traversal but differ in when they recurse into
// collections and when they allocate nil pointer-to-struct fields.
type walkPolicy struct {
	// DiveCollection reports whether a slice, array, or map node should be
	// traversed into its existing elements.
	DiveCollection func(ctx walkContext, field reflect.Value) bool

	// AllocPtrStruct reports whether a nil pointer-to-struct field should be
	// allocated before traversing its children.
	AllocPtrStruct func(ctx walkContext, field reflect.Value) bool
}

// walkAction is called for each resolved concrete field before child traversal.
type walkAction func(ctx walkContext, field reflect.Value) error

// walkSchema walks root using the compiled schema tree and calls action for each
// resolved field.
//
// Recursive type references are followed at runtime. activePointers prevents
// traversal from following an actual pointer cycle in the concrete object.
func walkSchema(
	root reflect.Value,
	rootNode *schema.Node,
	envPath []string,
	policy walkPolicy,
	action walkAction,
) error {
	if rootNode == nil {
		return nil
	}

	activePointers := make(map[uintptr]bool)

	for _, child := range rootNode.Children {
		childEnvPath, childEnvEnabled := applyWalkNodeEnvPath(
			envPath,
			true,
			child,
		)

		ctx := walkContext{
			Node:       child,
			Path:       nodePath(child),
			EnvPath:    childEnvPath,
			EnvEnabled: childEnvEnabled,
		}

		if err := walkNode(
			root,
			root,
			child,
			ctx,
			policy,
			action,
			activePointers,
		); err != nil {
			return err
		}
	}

	return nil
}

// walkNode resolves node against parent using the node's compiled index.
func walkNode(
	root reflect.Value,
	parent reflect.Value,
	node *schema.Node,
	ctx walkContext,
	policy walkPolicy,
	action walkAction,
	activePointers map[uintptr]bool,
) error {
	return walkNodeByIndex(
		root,
		parent,
		node,
		node.Index,
		ctx,
		policy,
		action,
		activePointers,
	)
}

// walkNodeByIndex resolves node using the provided index, calls action, and then
// recursively walks child values when policy allows it.
//
// The explicit index is needed when traversing collection elements or following
// recursive schema references because those fields must be resolved relative to
// the current concrete struct rather than the original root object.
func walkNodeByIndex(
	root reflect.Value,
	parent reflect.Value,
	node *schema.Node,
	index []int,
	ctx walkContext,
	policy walkPolicy,
	action walkAction,
	activePointers map[uintptr]bool,
) error {
	field, ok := fieldByIndex(parent, index)
	if !ok {
		return nil
	}

	if action != nil {
		if err := action(ctx, field); err != nil {
			return err
		}
	}

	if len(node.Children) == 0 &&
		node.Reference == nil &&
		!isCollectionNode(node) {
		return nil
	}

	trackedPointer, alreadyActive := beginPointerVisit(
		field,
		activePointers,
	)
	if alreadyActive {
		return nil
	}

	if trackedPointer != 0 {
		defer delete(activePointers, trackedPointer)
	}

	return walkChildren(
		root,
		field,
		node,
		ctx,
		policy,
		action,
		activePointers,
	)
}

// beginPointerVisit tracks a non-nil pointer-to-struct for one recursive
// branch. Shared objects reached through another root path are revisited after
// the current branch unwinds.
func beginPointerVisit(
	value reflect.Value,
	activePointers map[uintptr]bool,
) (pointer uintptr, alreadyActive bool) {
	value = unwrapInterface(value)
	if !value.IsValid() ||
		value.Kind() != reflect.Ptr ||
		value.IsNil() {
		return 0, false
	}

	if value.Type().Elem().Kind() != reflect.Struct {
		return 0, false
	}

	pointer = value.Pointer()
	if pointer == 0 {
		return 0, false
	}

	if activePointers[pointer] {
		return pointer, true
	}

	activePointers[pointer] = true

	return pointer, false
}

// walkChildren dispatches child traversal according to the concrete value kind.
func walkChildren(
	root reflect.Value,
	field reflect.Value,
	node *schema.Node,
	ctx walkContext,
	policy walkPolicy,
	action walkAction,
	activePointers map[uintptr]bool,
) error {
	field = unwrapInterface(field)
	if !field.IsValid() {
		return nil
	}

	switch field.Kind() {
	case reflect.Ptr:
		return walkPtrChildren(
			root,
			field,
			node,
			ctx,
			policy,
			action,
			activePointers,
		)

	case reflect.Struct:
		return walkStructChildren(
			root,
			field,
			node,
			ctx,
			policy,
			action,
			activePointers,
		)

	case reflect.Slice, reflect.Array:
		return walkSliceArrayChildren(
			field,
			node,
			ctx,
			policy,
			action,
			activePointers,
		)

	case reflect.Map:
		return walkMapChildren(
			field,
			node,
			ctx,
			policy,
			action,
			activePointers,
		)

	default:
		return nil
	}
}

// walkPtrChildren optionally allocates nil pointer-to-struct values and then
// continues traversal through the pointed struct.
func walkPtrChildren(
	root reflect.Value,
	field reflect.Value,
	node *schema.Node,
	ctx walkContext,
	policy walkPolicy,
	action walkAction,
	activePointers map[uintptr]bool,
) error {
	if field.IsNil() {
		if field.Type().Elem().Kind() != reflect.Struct {
			return nil
		}

		if policy.AllocPtrStruct == nil ||
			!policy.AllocPtrStruct(ctx, field) ||
			!field.CanSet() {
			return nil
		}

		field.Set(reflect.New(field.Type().Elem()))
	}

	return walkChildren(
		root,
		field.Elem(),
		node,
		ctx,
		policy,
		action,
		activePointers,
	)
}

// walkStructChildren walks ordinary nested struct children.
//
// Ordinary children use indexes rooted at root. A recursive reference reuses
// another node's children, whose direct field indexes must instead be resolved
// against the current concrete struct value.
func walkStructChildren(
	root reflect.Value,
	field reflect.Value,
	node *schema.Node,
	ctx walkContext,
	policy walkPolicy,
	action walkAction,
	activePointers map[uintptr]bool,
) error {
	if isDurationType(field.Type()) {
		return nil
	}

	children := node.Children
	parent := root
	relative := false

	if len(children) == 0 && node.Reference != nil {
		children = node.Reference.Children
		parent = field
		relative = true
	}

	for _, child := range children {
		childCtx := childWalkContext(ctx, child)

		index := child.Index
		if relative {
			index = directFieldIndex(child)
		}

		if err := walkNodeByIndex(
			root,
			parent,
			child,
			index,
			childCtx,
			policy,
			action,
			activePointers,
		); err != nil {
			return err
		}
	}

	return nil
}

// walkSliceArrayChildren walks existing slice and array elements when the policy
// allows collection traversal.
func walkSliceArrayChildren(
	collection reflect.Value,
	node *schema.Node,
	ctx walkContext,
	policy walkPolicy,
	action walkAction,
	activePointers map[uintptr]bool,
) error {
	if policy.DiveCollection == nil ||
		!policy.DiveCollection(ctx, collection) {
		return nil
	}

	children := schemaNodeChildren(node)
	if len(children) == 0 {
		for i := 0; i < collection.Len(); i++ {
			if action == nil {
				continue
			}

			elemCtx := collectionElementContext(
				ctx,
				fmt.Sprint(i),
			)
			if err := action(elemCtx, collection.Index(i)); err != nil {
				return err
			}
		}

		return nil
	}

	for i := 0; i < collection.Len(); i++ {
		elemCtx := collectionElementContext(
			ctx,
			fmt.Sprint(i),
		)

		if err := walkCollectionElement(
			collection.Index(i),
			children,
			elemCtx,
			policy,
			action,
			activePointers,
		); err != nil {
			return err
		}
	}

	return nil
}

// walkMapChildren walks existing map values when the policy allows collection
// traversal.
//
// Since map values are not settable, each value is copied, modified, and then
// written back into the map.
func walkMapChildren(
	mapValue reflect.Value,
	node *schema.Node,
	ctx walkContext,
	policy walkPolicy,
	action walkAction,
	activePointers map[uintptr]bool,
) error {
	if mapValue.IsNil() {
		return nil
	}

	if policy.DiveCollection == nil ||
		!policy.DiveCollection(ctx, mapValue) {
		return nil
	}

	children := schemaNodeChildren(node)

	for _, key := range mapValue.MapKeys() {
		value := unwrapInterface(mapValue.MapIndex(key))
		if !value.IsValid() {
			continue
		}

		updated := reflect.New(value.Type()).Elem()
		updated.Set(value)

		elemCtx := collectionElementContext(
			ctx,
			fmt.Sprint(key.Interface()),
		)

		if len(children) == 0 {
			if action != nil {
				if err := action(elemCtx, updated); err != nil {
					return err
				}
			}
		} else {
			if err := walkCollectionElement(
				updated,
				children,
				elemCtx,
				policy,
				action,
				activePointers,
			); err != nil {
				return err
			}
		}

		mapValue.SetMapIndex(key, updated)
	}

	return nil
}

// walkCollectionElement traverses one concrete struct or pointer-to-struct
// collection element.
//
// Pointer elements participate in the same active-pointer tracking used for
// ordinary pointer fields, preventing cycles such as a node containing a slice
// that points back to the node itself.
func walkCollectionElement(
	value reflect.Value,
	children []*schema.Node,
	ctx walkContext,
	policy walkPolicy,
	action walkAction,
	activePointers map[uintptr]bool,
) error {
	value = unwrapInterface(value)
	if !value.IsValid() {
		return nil
	}

	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil
		}

		pointer := value.Pointer()
		if pointer != 0 {
			if activePointers[pointer] {
				return nil
			}

			activePointers[pointer] = true
			defer delete(activePointers, pointer)
		}

		value = unwrapInterface(value.Elem())
	}

	if !value.IsValid() ||
		value.Kind() != reflect.Struct ||
		isDurationType(value.Type()) {
		return nil
	}

	for _, child := range children {
		childCtx := childWalkContext(ctx, child)

		if err := walkNodeByIndex(
			value,
			value,
			child,
			directFieldIndex(child),
			childCtx,
			policy,
			action,
			activePointers,
		); err != nil {
			return err
		}
	}

	return nil
}

// schemaNodeChildren returns the directly compiled children or, for a recursive
// schema node, the children of its referenced type definition.
func schemaNodeChildren(node *schema.Node) []*schema.Node {
	if node == nil {
		return nil
	}

	if len(node.Children) > 0 {
		return node.Children
	}

	if node.Reference != nil {
		return node.Reference.Children
	}

	return nil
}

// directFieldIndex returns the field index relative to the struct represented
// by the node's immediate parent. It is used when following recursive schema
// references and when traversing collection element structs.
func directFieldIndex(node *schema.Node) []int {
	if node == nil || len(node.Index) == 0 {
		return nil
	}

	return node.Index[len(node.Index)-1:]
}

// childWalkContext returns the runtime context for child below parent.
func childWalkContext(
	parent walkContext,
	child *schema.Node,
) walkContext {
	childEnvPath, childEnvEnabled := applyWalkNodeEnvPath(
		parent.EnvPath,
		parent.EnvEnabled,
		child,
	)

	return walkContext{
		Node:       child,
		Path:       joinRuntimePath(parent.Path, nodeLastName(child)),
		EnvPath:    childEnvPath,
		EnvEnabled: childEnvEnabled,
	}
}

// collectionElementContext returns the runtime context for one concrete
// collection element or map value.
//
// The schema collection marker [] is replaced by the concrete element index or
// map key.
func collectionElementContext(
	parent walkContext,
	key string,
) walkContext {
	envPath := parent.EnvPath
	if parent.EnvEnabled {
		envPath = appendEnvPart(parent.EnvPath, key)
	}

	path := parent.Path
	path = strings.TrimSuffix(path, "[]")

	return walkContext{
		Node:       parent.Node,
		Path:       path + "[" + key + "]",
		EnvPath:    envPath,
		EnvEnabled: parent.EnvEnabled,
	}
}

// nodePath returns the schema path for node using dot-separated name segments.
func nodePath(node *schema.Node) string {
	if node == nil {
		return ""
	}

	return strings.Join(node.Name, ".")
}

// nodeLastName returns the final public name segment for node.
func nodeLastName(node *schema.Node) string {
	if node == nil || len(node.Name) == 0 {
		return ""
	}

	return node.Name[len(node.Name)-1]
}

// joinRuntimePath joins parent and child runtime path segments.
func joinRuntimePath(parent, child string) string {
	if parent == "" {
		return child
	}

	if child == "" {
		return parent
	}

	return parent + "." + child
}

// applyWalkNodeEnvPath appends the node's effective environment name to parent.
//
// An explicit env:"-" disables the node and all descendants. Once disabled,
// environment traversal remains disabled even when a descendant defines an env
// tag.
func applyWalkNodeEnvPath(
	parent []string,
	parentEnabled bool,
	node *schema.Node,
) ([]string, bool) {
	if node == nil || !parentEnabled {
		return parent, false
	}

	part := ""
	if len(node.Env) > 0 {
		part = node.Env[len(node.Env)-1]
	}

	if part == "-" {
		return parent, false
	}

	if part == "" {
		part = jsonTagName(node.JSONTag)
	}

	if part == "-" {
		return parent, false
	}

	if part == "" {
		part = nodeLastName(node)
	}

	if part == "" {
		return parent, false
	}

	return appendEnvPart(parent, part), true
}

// jsonTagName returns the JSON field name, excluding comma-separated options.
func jsonTagName(tag string) string {
	if index := strings.IndexByte(tag, ','); index >= 0 {
		return tag[:index]
	}

	return tag
}

func appendEnvPart(parent []string, part string) []string {
	out := make([]string, 0, len(parent)+1)
	out = append(out, parent...)
	out = append(out, part)

	return out
}

// canSetLiteralValue reports whether value can receive a scalar literal through
// setLiteralValue.
//
// Pointer-to-scalar fields are supported because setLiteralValue allocates them
// when necessary.
func canSetLiteralValue(value reflect.Value) bool {
	if !value.IsValid() {
		return false
	}

	return isSupportedLiteralType(value.Type())
}

// isSupportedLiteralType reports whether typ is a scalar type supported by
// setLiteralValue, optionally wrapped in pointers.
func isSupportedLiteralType(typ reflect.Type) bool {
	if typ == nil {
		return false
	}

	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if isDurationType(typ) {
		return true
	}

	switch typ.Kind() {
	case reflect.String,
		reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr,
		reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128:
		return true

	default:
		return false
	}
}

// joinEnvPath joins environment path segments using underscores and normalizes
// the resulting variable name to upper case. Empty segments are ignored.
func joinEnvPath(path []string) string {
	parts := make([]string, 0, len(path))

	for _, part := range path {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, "_")

		if part == "" {
			continue
		}

		parts = append(parts, strings.ToUpper(part))
	}

	return strings.Join(parts, "_")
}

func envPrefixPath(prefix string) []string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return nil
	}

	prefix = strings.Trim(prefix, "_")
	if prefix == "" {
		return nil
	}

	parts := strings.Split(prefix, "_")
	path := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			path = append(path, part)
		}
	}

	return path
}
