package constants

// Namespace is the shared logical namespace used for model errors and keys.
const Namespace = "model"

const (
	// RuleMin is the canonical built-in rule name for minimum constraints.
	RuleMin = "min"
	// RuleMax is the canonical built-in rule name for maximum constraints.
	RuleMax = "max"
	// RuleNonzero is the canonical built-in rule name for non-zero constraints.
	RuleNonzero = "nonzero"
	// RuleOneOf is the canonical built-in rule name for allowed-set membership checks.
	RuleOneOf = "oneof"
	// RuleEmail is the canonical built-in rule name for email validation.
	RuleEmail = "email"
	// RuleUUID is the canonical built-in rule name for UUID validation.
	RuleUUID = "uuid"
)

const (
	// RuleStringMin is the backward-compatible alias for the string min rule name.
	RuleStringMin = RuleMin
	// RuleStringMax is the backward-compatible alias for the string max rule name.
	RuleStringMax = RuleMax
	// RuleIntMin is the backward-compatible alias for the int min rule name.
	RuleIntMin = RuleMin
	// RuleIntMax is the backward-compatible alias for the int max rule name.
	RuleIntMax = RuleMax
	// RuleInt64Min is the backward-compatible alias for the int64 min rule name.
	RuleInt64Min = RuleMin
	// RuleInt64Max is the backward-compatible alias for the int64 max rule name.
	RuleInt64Max = RuleMax
	// RuleFloat64Min is the backward-compatible alias for the float64 min rule name.
	RuleFloat64Min = RuleMin
	// RuleFloat64Max is the backward-compatible alias for the float64 max rule name.
	RuleFloat64Max = RuleMax
)
