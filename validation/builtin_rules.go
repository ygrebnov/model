package validation

import (
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/ygrebnov/errorc"

	modelerrors "github.com/ygrebnov/model/errors"
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

// string rules
// min(length): requires one integer parameter. If missing -> error. If <1 -> noop.
func getStrMinRule() (Rule, error) {
	return NewRule[string]("min", func(s string, params ...string) error {
		if len(params) == 0 {
			return errorc.With(
				modelerrors.ErrRuleMissingParameter,
				errorc.String(modelerrors.ErrorFieldRuleName, "min"),
			)
		}
		raw := strings.TrimSpace(params[0])
		v, err := strconv.ParseInt(raw, 10, 0)
		if err != nil {
			return errorc.With(
				modelerrors.ErrRuleInvalidParameter,
				errorc.String(modelerrors.ErrorFieldRuleName, "min"),
				errorc.String(modelerrors.ErrorFieldRuleParamName, "length"),
				errorc.String(modelerrors.ErrorFieldRuleParamValue, raw),
				errorc.Error(modelerrors.ErrorFieldCause, err),
			)
		}
		if v < 1 { // noop as requested
			return nil
		}
		if int(v) > len(s) { // length too small
			return errorc.With(
				modelerrors.ErrRuleConstraintViolated,
				errorc.String(modelerrors.ErrorFieldRuleName, "min"),
				errorc.String(modelerrors.ErrorFieldRuleParamName, "length"),
				errorc.String(modelerrors.ErrorFieldRuleParamValue, raw),
			)
		}
		return nil
	})
}

// email rule: deliberately simple; not RFC 5322 exhaustive. Provides lightweight validation.
func getStrEmailRule() (Rule, error) {
	return NewRule[string]("email", func(s string, _ ...string) error {
		if s == "" { // treat empty as error, keeping semantics similar to prior nonempty
			return errorc.With(
				modelerrors.ErrRuleConstraintViolated,
				errorc.String(modelerrors.ErrorFieldRuleName, "email"),
			)
		}
		if strings.Count(s, "@") != 1 {
			return errorc.With(
				modelerrors.ErrRuleConstraintViolated,
				errorc.String(modelerrors.ErrorFieldRuleName, "email"),
				errorc.String(modelerrors.ErrorFieldRuleParamName, "at_count"),
				errorc.String(modelerrors.ErrorFieldRuleParamValue, "1"),
			)
		}
		parts := strings.Split(s, "@")
		local, domain := parts[0], parts[1]
		if local == "" || domain == "" {
			return errorc.With(
				modelerrors.ErrRuleConstraintViolated,
				errorc.String(modelerrors.ErrorFieldRuleName, "email"),
				errorc.String(modelerrors.ErrorFieldRuleParamName, "local_domain_nonempty"),
			)
		}
		if strings.ContainsAny(s, " \t\n\r") {
			return errorc.With(
				modelerrors.ErrRuleConstraintViolated,
				errorc.String(modelerrors.ErrorFieldRuleName, "email"),
				errorc.String(modelerrors.ErrorFieldRuleParamName, "no_whitespace"),
			)
		}
		if !strings.Contains(domain, ".") { // simple domain heuristic
			return errorc.With(
				modelerrors.ErrRuleConstraintViolated,
				errorc.String(modelerrors.ErrorFieldRuleName, "email"),
				errorc.String(modelerrors.ErrorFieldRuleParamName, "domain_has_dot"),
			)
		}
		return nil
	})
}

// oneof rule: value must match one of the provided parameters.
func getStrOneofRule() (Rule, error) {
	return NewRule[string]("oneof", func(s string, params ...string) error {
		if len(params) == 0 {
			return errorc.With(
				modelerrors.ErrRuleMissingParameter,
				errorc.String(modelerrors.ErrorFieldRuleName, "oneof"),
			)
		}
		for _, p := range params {
			if s == p {
				return nil
			}
		}
		return errorc.With(
			modelerrors.ErrRuleConstraintViolated,
			errorc.String(modelerrors.ErrorFieldRuleName, "oneof"),
			// we expose the allowed set as the param value for debugging/inspection
			errorc.String(modelerrors.ErrorFieldRuleParamName, "allowed"),
			errorc.String(modelerrors.ErrorFieldRuleParamValue, strings.Join(params, ",")),
		)
	})
}

// int rules
// positive: n must be > 0
func getIntPositiveRule() (Rule, error) {
	return NewRule[int]("positive", func(n int, _ ...string) error {
		if n <= 0 {
			return errorc.With(
				modelerrors.ErrRuleConstraintViolated,
				errorc.String(modelerrors.ErrorFieldRuleName, "positive"),
			)
		}
		return nil
	})
}

// nonzero: n must not be zero
func getIntNonzeroRule() (Rule, error) {
	return NewRule[int]("nonzero", func(n int, _ ...string) error {
		if n == 0 {
			return errorc.With(
				modelerrors.ErrRuleConstraintViolated,
				errorc.String(modelerrors.ErrorFieldRuleName, "nonzero"),
			)
		}
		return nil
	})
}

// oneof: n must equal one of the provided integer parameters
func getIntOneofRule() (Rule, error) {
	return NewRule[int]("oneof", func(n int, params ...string) error {
		if len(params) == 0 {
			return errorc.With(
				modelerrors.ErrRuleMissingParameter,
				errorc.String(modelerrors.ErrorFieldRuleName, "oneof"),
			)
		}
		for _, p := range params {
			raw := strings.TrimSpace(p)
			v, err := strconv.ParseInt(raw, 10, 0)
			if err != nil {
				return errorc.With(
					modelerrors.ErrRuleInvalidParameter,
					errorc.String(modelerrors.ErrorFieldRuleName, "oneof"),
					errorc.String(modelerrors.ErrorFieldRuleParamName, "value"),
					errorc.String(modelerrors.ErrorFieldRuleParamValue, raw),
					errorc.Error(modelerrors.ErrorFieldCause, err),
				)
			}
			if int(v) == n {
				return nil
			}
		}
		return errorc.With(
			modelerrors.ErrRuleConstraintViolated,
			errorc.String(modelerrors.ErrorFieldRuleName, "oneof"),
			errorc.String(modelerrors.ErrorFieldRuleParamName, "allowed"),
			errorc.String(modelerrors.ErrorFieldRuleParamValue, strings.Join(params, ",")),
		)
	})
}

// int64 rules
func getInt64PositiveRule() (Rule, error) {
	return NewRule[int64]("positive", func(n int64, _ ...string) error {
		if n <= 0 {
			return errorc.With(
				modelerrors.ErrRuleConstraintViolated,
				errorc.String(modelerrors.ErrorFieldRuleName, "positive"),
			)
		}
		return nil
	})
}

func getInt64NonzeroRule() (Rule, error) {
	return NewRule[int64]("nonzero", func(n int64, _ ...string) error {
		if n == 0 {
			return errorc.With(
				modelerrors.ErrRuleConstraintViolated,
				errorc.String(modelerrors.ErrorFieldRuleName, "nonzero"),
			)
		}
		return nil
	})
}

func getInt64OneofRule() (Rule, error) {
	return NewRule[int64]("oneof", func(n int64, params ...string) error {
		if len(params) == 0 {
			return errorc.With(
				modelerrors.ErrRuleMissingParameter,
				errorc.String(modelerrors.ErrorFieldRuleName, "oneof"),
			)
		}
		for _, p := range params {
			raw := strings.TrimSpace(p)
			v, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				return errorc.With(
					modelerrors.ErrRuleInvalidParameter,
					errorc.String(modelerrors.ErrorFieldRuleName, "oneof"),
					errorc.String(modelerrors.ErrorFieldRuleParamName, "value"),
					errorc.String(modelerrors.ErrorFieldRuleParamValue, raw),
					errorc.Error(modelerrors.ErrorFieldCause, err),
				)
			}
			if v == n {
				return nil
			}
		}
		return errorc.With(
			modelerrors.ErrRuleConstraintViolated,
			errorc.String(modelerrors.ErrorFieldRuleName, "oneof"),
			errorc.String(modelerrors.ErrorFieldRuleParamName, "allowed"),
			errorc.String(modelerrors.ErrorFieldRuleParamValue, strings.Join(params, ",")),
		)
	})
}

// float64 rules
func getFloat64PositiveRule() (Rule, error) {
	return NewRule[float64]("positive", func(n float64, _ ...string) error {
		if !(n > 0) {
			return errorc.With(
				modelerrors.ErrRuleConstraintViolated,
				errorc.String(modelerrors.ErrorFieldRuleName, "positive"),
			)
		}
		return nil
	})
}

func getFloat64NonzeroRule() (Rule, error) {
	return NewRule[float64]("nonzero", func(n float64, _ ...string) error {
		if n == 0 {
			return errorc.With(
				modelerrors.ErrRuleConstraintViolated,
				errorc.String(modelerrors.ErrorFieldRuleName, "nonzero"),
			)
		}
		return nil
	})
}

func getFloat64OneofRule() (Rule, error) {
	return NewRule[float64]("oneof", func(n float64, params ...string) error {
		if len(params) == 0 {
			return errorc.With(
				modelerrors.ErrRuleMissingParameter,
				errorc.String(modelerrors.ErrorFieldRuleName, "oneof"),
			)
		}
		for _, p := range params {
			raw := strings.TrimSpace(p)
			v, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				return errorc.With(
					modelerrors.ErrRuleInvalidParameter,
					errorc.String(modelerrors.ErrorFieldRuleName, "oneof"),
					errorc.String(modelerrors.ErrorFieldRuleParamName, "value"),
					errorc.String(modelerrors.ErrorFieldRuleParamValue, raw),
					errorc.Error(modelerrors.ErrorFieldCause, err),
				)
			}
			if v == n {
				return nil
			}
		}
		return errorc.With(
			modelerrors.ErrRuleConstraintViolated,
			errorc.String(modelerrors.ErrorFieldRuleName, "oneof"),
			errorc.String(modelerrors.ErrorFieldRuleParamName, "allowed"),
			errorc.String(modelerrors.ErrorFieldRuleParamValue, strings.Join(params, ",")),
		)
	})
}

// ensureBuiltIns initializes built-in rules exactly once.
func ensureBuiltIns() {
	builtInsOnce.Do(func() {
		builtInMap = make(map[key]Rule)

		// string rules
		strMin, _ := getStrMinRule()
		strEmail, _ := getStrEmailRule()
		strOneof, _ := getStrOneofRule()
		builtinStringRules = []Rule{strMin, strEmail, strOneof}

		// int rules
		positiveInt, _ := getIntPositiveRule()
		nonzeroInt, _ := getIntNonzeroRule()
		oneofInt, _ := getIntOneofRule()
		builtinIntRules = []Rule{positiveInt, nonzeroInt, oneofInt}

		// int64 rules
		positiveInt64, _ := getInt64PositiveRule()
		nonzeroInt64, _ := getInt64NonzeroRule()
		oneofInt64, _ := getInt64OneofRule()
		builtinInt64Rules = []Rule{positiveInt64, nonzeroInt64, oneofInt64}

		// float64 rules
		positiveFloat64, _ := getFloat64PositiveRule()
		nonzeroFloat64, _ := getFloat64NonzeroRule()
		oneofFloat64, _ := getFloat64OneofRule()
		builtinFloat64Rules = []Rule{positiveFloat64, nonzeroFloat64, oneofFloat64}

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
