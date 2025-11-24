package errors

import (
	"github.com/ygrebnov/errorc"

	"github.com/ygrebnov/model/constants"
)

var namespace = errorc.Namespace(constants.Namespace)

// Sentinel errors for constructor misuses. Use errors.Is to match.
var (
	ErrNilObject                     = namespace.NewError("nil object")
	ErrNotStructPtr                  = namespace.NewError("object must be a non-nil pointer to struct")
	ErrInvalidRule                   = namespace.NewError("rule must have non-empty name and non-nil function")
	ErrRuleTypeMismatch              = namespace.NewError("rule type mismatch")
	ErrDuplicateOverloadRule         = namespace.NewError("duplicate overload rule")
	ErrRuleNotFound                  = namespace.NewError("rule not found")
	ErrRuleOverloadNotFound          = namespace.NewError("rule overload not found")
	ErrInvalidValue                  = namespace.NewError("invalid value")
	ErrAmbiguousRule                 = namespace.NewError("ambiguous rule")
	ErrSetDefault                    = namespace.NewError("cannot set default value")
	ErrDefaultLiteralUnsupportedKind = namespace.NewError("default literal unsupported kind")

	// Validation rule argument and parameter errors
	ErrRuleMissingParameter   = namespace.NewError("rule parameter is required")
	ErrRuleInvalidParameter   = namespace.NewError("rule parameter is invalid")
	ErrRuleConstraintViolated = namespace.NewError("rule constraint violated")

	// Test-only/sample rule errors (used in model_validate_test)
	ErrRuleMin1Failed       = namespace.NewError("min(1) rule failed")
	ErrRuleNonZeroDurFailed = namespace.NewError("nonZeroDur rule failed")
)

var newKey = errorc.KeyFactory(constants.ErrorFieldNamespace)

// Internal hierarchical segments used to build dotted keys.
const (
	keySegmentRule    = "rule"
	keySegmentDefault = "default"
	keySegmentField   = "field"
)

// Exported structured error field keys
var (
	ErrorFieldRuleName       = newKey("name", keySegmentRule)            // model.rule.name
	ErrorFieldFieldType      = newKey("field_type", keySegmentRule)      // model.rule.field_type
	ErrorFieldValueType      = newKey("value_type", keySegmentRule)      // model.rule.value_type
	ErrorFieldAvailableTypes = newKey("available_types", keySegmentRule) // model.rule.available_types
	ErrorFieldExactTypes     = newKey("exact_types", keySegmentRule)     // model.rule.exact_types (reserved)

	// Parameters/arguments for a rule invocation
	ErrorFieldRuleParamName  = newKey("param_name", keySegmentRule)  // model.rule.param_name
	ErrorFieldRuleParamValue = newKey("param_value", keySegmentRule) // model.rule.param_value
)

var (
	ErrorFieldDefaultLiteralKind = newKey("literal_kind", keySegmentDefault) // model.default.literal.kind
)

var (
	ErrorFieldFieldName = newKey("name", keySegmentField) // model.field.name
)

var (
	ErrorFieldObjectType = newKey("object_type")
	ErrorFieldPhase      = newKey("phase")
	ErrorFieldCause      = newKey("cause")
)
