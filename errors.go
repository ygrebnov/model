package model

import "errors"

const Namespace = "model"

// Sentinel errors for constructor misuses. Use errors.Is to match.
var (
	ErrNilObject             = errors.New(Namespace + ": nil object")
	ErrNotStructPtr          = errors.New(Namespace + ": object must be a pointer to struct")
	ErrDuplicateOverloadRule = errors.New(Namespace + ": duplicate overload rule")
	ErrRuleNotFound          = errors.New(Namespace + ": rule not found")
	ErrRuleOverloadNotFound  = errors.New(Namespace + ": rule overload not found")
	ErrInvalidValue          = errors.New(Namespace + ": invalid value")
	ErrAmbiguousRule         = errors.New(Namespace + ": ambiguous rule")
)

// ErrorField is a strongly-typed key used for structured error context (e.g. Kibana / log filtering).
// Underlying type is string so it can be used directly with errorc.Field (which accepts any underlying string type).
type ErrorField string

// Namespace for all exported error field keys (informational; not currently prefixed automatically).
const ErrorFieldNamespace = "model"

// Exported structured error field keys. Keep string values stable for log queries.
const (
	ErrorFieldRuleName       ErrorField = "rule_name"
	ErrorFieldFieldType      ErrorField = "field_type"
	ErrorFieldValueType      ErrorField = "value_type"
	ErrorFieldAvailableTypes ErrorField = "available_types"
	ErrorFieldExactTypes     ErrorField = "exact_types" // reserved for future ambiguity details
)
