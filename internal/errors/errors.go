package errors

import "errors"

const Namespace = "model"

// Sentinel errors for constructor misuses. Use errors.Is to match.
var (
	ErrNilObject                     = errors.New(Namespace + ": nil object")
	ErrNotStructPtr                  = errors.New(Namespace + ": object must be a non-nil pointer to struct")
	ErrDuplicateOverloadRule         = errors.New(Namespace + ": duplicate overload rule")
	ErrRuleNotFound                  = errors.New(Namespace + ": rule not found")
	ErrRuleOverloadNotFound          = errors.New(Namespace + ": rule overload not found")
	ErrInvalidValue                  = errors.New(Namespace + ": invalid value")
	ErrAmbiguousRule                 = errors.New(Namespace + ": ambiguous rule")
	ErrSetDefault                    = errors.New(Namespace + ": cannot set default value")
	ErrDefaultLiteralUnsupportedKind = errors.New(Namespace + ": default literal unsupported kind")
)

// ErrorField is a strongly-typed key used for structured error context (e.g. Kibana / log filtering).
type ErrorField string

// ErrorFieldNamespace for all exported error field keys.
const ErrorFieldNamespace = Namespace

// Internal hierarchical segments used to build dotted keys.
const (
	_errorFieldRuleSegment    = ".rule."
	_errorFieldDefaultSegment = ".default"
	_errorFieldFieldSegment   = ".field."
)

// Exported structured error field keys
const (
	ErrorFieldRuleName       ErrorField = ErrorFieldNamespace + _errorFieldRuleSegment + "name"            // model.rule.name
	ErrorFieldFieldType      ErrorField = ErrorFieldNamespace + _errorFieldRuleSegment + "field_type"      // model.rule.field_type
	ErrorFieldValueType      ErrorField = ErrorFieldNamespace + _errorFieldRuleSegment + "value_type"      // model.rule.value_type
	ErrorFieldAvailableTypes ErrorField = ErrorFieldNamespace + _errorFieldRuleSegment + "available_types" // model.rule.available_types
	ErrorFieldExactTypes     ErrorField = ErrorFieldNamespace + _errorFieldRuleSegment + "exact_types"     // model.rule.exact_types (reserved)
)

const (
	ErrorFieldDefaultLiteralKind ErrorField = ErrorFieldNamespace + _errorFieldDefaultSegment + ".literal_kind" // model.default.literal.kind
)

const (
	ErrorFieldFieldName ErrorField = ErrorFieldNamespace + _errorFieldFieldSegment + "name" // model.field.name
)

const (
	ErrorFieldObjectType ErrorField = ErrorFieldNamespace + ".object_type"
	ErrorFieldPhase      ErrorField = ErrorFieldNamespace + ".phase"
	ErrorFieldCause      ErrorField = ErrorFieldNamespace + ".cause"
)
