package keys

import (
	"github.com/ygrebnov/keys"
)

const (
	keySegmentRule  = "rule"
	keySegmentParam = "param"
	keySegmentField = "field"
)

var (
	newRuleKey      = keys.Factory(keys.WithSegments(keySegmentRule))
	newRuleParamKey = keys.Factory(keys.WithSegments(keySegmentRule, keySegmentParam))
	newFieldKey     = keys.Factory(keys.WithSegments(keySegmentField))
)

var (
	// RuleName identifies the rule name in structured error context.
	RuleName = newRuleKey("name")
	// RuleParamName identifies a rule parameter name in structured error context.
	RuleParamName = newRuleParamKey("name")
	// RuleParamValue identifies a rule parameter value in structured error context.
	RuleParamValue = newRuleParamKey("value")

	FieldName = newFieldKey("name")
	FieldPath = newFieldKey("path")
	// FieldType identifies a field type in structured error context.
	FieldType = newFieldKey("type")
	// FieldAvailableTypes identifies the available field types for rule overload diagnostics.
	FieldAvailableTypes = newFieldKey("available_types")

	// ValueType identifies the runtime value type in structured error context.
	ValueType = keys.New("value.type")
	// ObjectType identifies the runtime object type in structured error context.
	ObjectType   = keys.New("object.type")
	ExpectedType = keys.New("expected_type")
	// DefaultLiteralKind identifies the kind used when parsing default literal values.
	DefaultLiteralKind = keys.New("default.literal.kind")

	TagDefault = keys.New("tag.default")
	// Phase identifies the operation phase in structured error context.
	Phase = keys.New("phase")
	// Cause identifies an underlying cause in structured error context.
	Cause = keys.New("cause")
	Value = keys.New("value")
)
