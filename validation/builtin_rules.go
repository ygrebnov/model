package validation

import (
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/ygrebnov/errorc"
	"github.com/ygrebnov/model/constants"
	modelerrors "github.com/ygrebnov/model/errors"
	"github.com/ygrebnov/model/keys"
)

// Built-ins are always implicitly available.

// key consists of a name and a field value type.
type key struct {
	name      string
	fieldType reflect.Type
}

// Lazy built-in rule storage.
var (
	builtInsOnce        sync.Once
	builtInMap          map[key]Rule
	builtinStringRules  []Rule
	builtinIntRules     []Rule
	builtinInt64Rules   []Rule
	builtinFloat64Rules []Rule
)

const (
	paramNameAllowed = "allowed"
	paramNameLength  = "length"
	paramNameValue   = "value"

	emailCheckAtCount             = "at_count"
	emailCheckDomainHasDot        = "domain_has_dot"
	emailCheckLocalDomainNonempty = "local_domain_nonempty"
	emailCheckNoWhitespace        = "no_whitespace"

	uuidCheckFormat = "format"
	uuidCheckHex    = "hex"
	uuidLength      = 36
)

var uuidHyphenPositions = [4]int{8, 13, 18, 23}

func newRuleMissingParameterError(ruleName string) error {
	return errorc.With(
		modelerrors.ErrRuleMissingParameter,
		errorc.String(keys.RuleName, ruleName),
	)
}

func newRuleInvalidParameterError(ruleName, paramName, paramValue string, cause error) error {
	return errorc.With(
		modelerrors.ErrRuleInvalidParameter,
		errorc.String(keys.RuleName, ruleName),
		errorc.String(keys.RuleParamName, paramName),
		errorc.String(keys.RuleParamValue, paramValue),
		errorc.Error(keys.Cause, cause),
	)
}

func newRuleConstraintViolationError(ruleName string) error {
	return errorc.With(
		modelerrors.ErrRuleConstraintViolated,
		errorc.String(keys.RuleName, ruleName),
	)
}

func newRuleConstraintViolationWithStringParamError(ruleName, paramName, paramValue string) error {
	return errorc.With(
		modelerrors.ErrRuleConstraintViolated,
		errorc.String(keys.RuleName, ruleName),
		errorc.String(keys.RuleParamName, paramName),
		errorc.String(keys.RuleParamValue, paramValue),
	)
}

func newRuleConstraintViolationWithIntParamError(ruleName, paramName string, paramValue int) error {
	return errorc.With(
		modelerrors.ErrRuleConstraintViolated,
		errorc.String(keys.RuleName, ruleName),
		errorc.String(keys.RuleParamName, paramName),
		errorc.Int(keys.RuleParamValue, paramValue),
	)
}

func newRuleConstraintViolationWithParamNameError(ruleName, paramName string) error {
	return errorc.With(
		modelerrors.ErrRuleConstraintViolated,
		errorc.String(keys.RuleName, ruleName),
		errorc.String(keys.RuleParamName, paramName),
	)
}

func mustRule(r Rule, err error) Rule {
	if err != nil {
		panic(err)
	}

	return r
}

func splitEmailParts(s string) (local, domain string) {
	parts := strings.Split(s, "@")
	return parts[0], parts[1]
}

func validateBuiltinEmail(s string) error {
	if s == "" { // treat empty as error, keeping semantics similar to prior nonempty
		return newRuleConstraintViolationError(constants.RuleEmail)
	}
	if strings.Count(s, "@") != 1 {
		return newRuleConstraintViolationWithStringParamError(constants.RuleEmail, emailCheckAtCount, "1")
	}

	local, domain := splitEmailParts(s)
	if local == "" || domain == "" {
		return newRuleConstraintViolationWithParamNameError(constants.RuleEmail, emailCheckLocalDomainNonempty)
	}
	if strings.ContainsAny(s, " \t\n\r") {
		return newRuleConstraintViolationWithParamNameError(constants.RuleEmail, emailCheckNoWhitespace)
	}
	if !strings.Contains(domain, ".") { // simple domain heuristic
		return newRuleConstraintViolationWithParamNameError(constants.RuleEmail, emailCheckDomainHasDot)
	}

	return nil
}

func isUUIDHyphenPosition(index int) bool {
	for _, position := range uuidHyphenPositions {
		if index == position {
			return true
		}
	}

	return false
}

func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func validateBuiltinUUID(s string) error {
	// Empty is invalid; caller can omit the rule if empty is allowed.
	// Canonical form: 36 chars, 8-4-4-4-12 with hyphens, hex digits only.
	if len(s) != uuidLength {
		return newRuleConstraintViolationWithIntParamError(constants.RuleUUID, paramNameLength, len(s))
	}

	for i := 0; i < len(s); i++ {
		c := s[i]
		if isUUIDHyphenPosition(i) {
			if c != '-' {
				return newRuleConstraintViolationWithStringParamError(
					constants.RuleUUID, uuidCheckFormat, "expected hyphens at 8,13,18,23",
				)
			}
			continue
		}
		if !isHexDigit(c) {
			return newRuleConstraintViolationWithStringParamError(constants.RuleUUID, uuidCheckHex, "non-hex character")
		}
	}

	return nil
}

// string rules
func getStringMinMaxRule(
	name string,
	noop func(v int64) bool,
	compare func(a, b int) bool,
) (Rule, error) {
	return NewRule[string](name, func(s string, params ...string) error {
		if len(params) == 0 {
			return newRuleMissingParameterError(name)
		}
		raw := strings.TrimSpace(params[0])
		v, err := strconv.ParseInt(raw, 10, 0)
		if err != nil {
			return newRuleInvalidParameterError(name, paramNameLength, raw, err)
		}
		if noop(v) { // noop as requested
			return nil
		}
		if compare(int(v), len(s)) {
			return newRuleConstraintViolationWithStringParamError(name, paramNameLength, raw)
		}
		return nil
	})
}

// min(length): requires one integer parameter. If missing -> error. If <1 -> noop.
func getStrMinRule() (Rule, error) {
	return getStringMinMaxRule(
		constants.RuleStringMin,
		func(v int64) bool { return v < 1 },
		func(a, b int) bool { return a > b },
	)
}

// max(length): requires one integer parameter. If missing -> error. If <0 -> noop.
func getStrMaxRule() (Rule, error) {
	return getStringMinMaxRule(
		constants.RuleStringMax,
		func(v int64) bool { return v < 0 },
		func(a, b int) bool { return a < b },
	)
}

// email rule: deliberately simple; not RFC 5322 exhaustive. Provides lightweight validation.
func getStrEmailRule() (Rule, error) {
	return NewRule[string](constants.RuleEmail, func(s string, _ ...string) error {
		return validateBuiltinEmail(s)
	})
}

// oneof rule: value must match one of the provided parameters.
func getStrOneofRule() (Rule, error) {
	return NewRule[string](constants.RuleOneOf, func(s string, params ...string) error {
		if len(params) == 0 {
			return newRuleMissingParameterError(constants.RuleOneOf)
		}
		for _, p := range params {
			if s == p {
				return nil
			}
		}
		// we expose the allowed set as the param value for debugging/inspection
		return newRuleConstraintViolationWithStringParamError(
			constants.RuleOneOf,
			paramNameAllowed,
			strings.Join(params, ","),
		)
	})
}

// uuid rule: value must be a valid canonical UUID string (lower/upper hex, 8-4-4-4-12 format).
func getStrUUIDRule() (Rule, error) {
	return NewRule[string](constants.RuleUUID, func(s string, _ ...string) error {
		return validateBuiltinUUID(s)
	})
}

func getNumericMinMaxRule[T interface{ int | int64 | float64 }](
	name string,
	parse func(string) (T, error),
	compare func(a, b T) bool,
) (Rule, error) {
	return NewRule[T](name, func(n T, params ...string) error {
		if len(params) == 0 {
			return newRuleMissingParameterError(name)
		}
		raw := strings.TrimSpace(params[0])
		v, err := parse(raw)
		if err != nil {
			return newRuleInvalidParameterError(name, paramNameValue, raw, err)
		}
		if compare(n, v) {
			return newRuleConstraintViolationWithStringParamError(name, paramNameValue, raw)
		}
		return nil
	})
}

func getNumericNonzeroRule[T interface{ int | int64 | float64 }](name string) (Rule, error) {
	return NewRule[T](name, func(n T, _ ...string) error {
		if n == 0 {
			return newRuleConstraintViolationError(name)
		}
		return nil
	})
}

func getNumericOneofRule[T interface{ int | int64 | float64 }](
	name string,
	parse func(string) (T, error),
) (Rule, error) {
	return NewRule[T](name, func(n T, params ...string) error {
		if len(params) == 0 {
			return newRuleMissingParameterError(name)
		}
		for _, p := range params {
			raw := strings.TrimSpace(p)
			v, err := parse(raw)
			if err != nil {
				return newRuleInvalidParameterError(name, paramNameValue, raw, err)
			}
			if v == n {
				return nil
			}
		}
		return newRuleConstraintViolationWithStringParamError(name, paramNameAllowed, strings.Join(params, ","))
	})
}

// int rules
// min(value): requires one integer parameter. Field value must be >= param.
func getIntMinRule() (Rule, error) {
	return getNumericMinMaxRule[int](constants.RuleMin, strconv.Atoi, func(a, b int) bool { return a < b })
}

// max(value): requires one integer parameter. Field value must be <= param.
func getIntMaxRule() (Rule, error) {
	return getNumericMinMaxRule[int](constants.RuleMax, strconv.Atoi, func(a, b int) bool { return a > b })
}

// nonzero: n must not be zero
func getIntNonzeroRule() (Rule, error) {
	return getNumericNonzeroRule[int](constants.RuleNonzero)
}

// oneof: n must equal one of the provided integer parameters
func getIntOneofRule() (Rule, error) {
	return getNumericOneofRule[int](
		constants.RuleOneOf,
		strconv.Atoi,
	)
}

// int64 rules
// min(value): requires one integer parameter. Field value must be >= param.
func getInt64MinRule() (Rule, error) {
	return getNumericMinMaxRule[int64](
		constants.RuleMin,
		func(s string) (int64, error) { return strconv.ParseInt(s, 10, 64) },
		func(a, b int64) bool { return a < b },
	)
}

// max(value): requires one integer parameter. Field value must be <= param.
func getInt64MaxRule() (Rule, error) {
	return getNumericMinMaxRule[int64](
		constants.RuleMax,
		func(s string) (int64, error) { return strconv.ParseInt(s, 10, 64) },
		func(a, b int64) bool { return a > b },
	)
}

func getInt64NonzeroRule() (Rule, error) {
	return getNumericNonzeroRule[int64](constants.RuleNonzero)
}

func getInt64OneofRule() (Rule, error) {
	return getNumericOneofRule[int64](
		constants.RuleOneOf,
		func(s string) (int64, error) { return strconv.ParseInt(s, 10, 64) },
	)
}

// float64 rules
// min(value): requires one integer parameter. Field value must be >= param.
func getFloat64MinRule() (Rule, error) {
	return getNumericMinMaxRule[float64](
		constants.RuleMin,
		func(s string) (float64, error) { return strconv.ParseFloat(s, 64) },
		func(a, b float64) bool { return a < b },
	)
}

// max(value): requires one integer parameter. Field value must be <= param.
func getFloat64MaxRule() (Rule, error) {
	return getNumericMinMaxRule[float64](
		constants.RuleMax,
		func(s string) (float64, error) { return strconv.ParseFloat(s, 64) },
		func(a, b float64) bool { return a > b },
	)
}

func getFloat64NonzeroRule() (Rule, error) {
	return getNumericNonzeroRule[float64](constants.RuleNonzero)
}

func getFloat64OneofRule() (Rule, error) {
	return getNumericOneofRule[float64](
		constants.RuleOneOf,
		func(s string) (float64, error) { return strconv.ParseFloat(s, 64) },
	)
}

// ensureBuiltIns initializes built-in rules exactly once.
func ensureBuiltIns() {
	builtInsOnce.Do(func() {
		builtInMap = make(map[key]Rule)

		// string rules
		builtinStringRules = []Rule{
			mustRule(getStrMinRule()),
			mustRule(getStrMaxRule()),
			mustRule(getStrEmailRule()),
			mustRule(getStrOneofRule()),
			mustRule(getStrUUIDRule()),
		}

		// int rules
		builtinIntRules = []Rule{
			mustRule(getIntMinRule()),
			mustRule(getIntMaxRule()),
			mustRule(getIntNonzeroRule()),
			mustRule(getIntOneofRule()),
		}

		// int64 rules
		builtinInt64Rules = []Rule{
			mustRule(getInt64MinRule()),
			mustRule(getInt64MaxRule()),
			mustRule(getInt64NonzeroRule()),
			mustRule(getInt64OneofRule()),
		}

		// float64 rules
		builtinFloat64Rules = []Rule{
			mustRule(getFloat64MinRule()),
			mustRule(getFloat64MaxRule()),
			mustRule(getFloat64NonzeroRule()),
			mustRule(getFloat64OneofRule()),
		}

		// fill map
		register := func(rs []Rule) {
			for _, r := range rs {
				builtInMap[key{r.GetName(), r.getFieldType()}] = r
			}
		}
		register(builtinStringRules)
		register(builtinIntRules)
		register(builtinInt64Rules)
		register(builtinFloat64Rules)
	})
}

// lookupBuiltin returns a built-in rule by (Name,type) if present.
func lookupBuiltin(name string, t reflect.Type) (Rule, bool) {
	ensureBuiltIns()
	r, ok := builtInMap[key{name, t}]
	return r, ok
}
