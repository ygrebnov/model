package model

type options struct {
	envPrefix string
	rules     []Rule
}

// Option customizes Binding, DynamicBinding or wrappers behavior.
type Option func(o *options)

// WithEnvPrefix adds a non-empty environment variable name prefix for ApplyEnv
// and ValidateWithDefaults.
func WithEnvPrefix(prefix string) Option {
	return func(o *options) {
		if prefix != "" {
			o.envPrefix = prefix
		}
	}
}

// WithRules registers one or many named custom validation rules during binding construction.
//
// See the Rule type and NewRule function for details on creating rules.
func WithRules(rules ...Rule) Option {
	return func(o *options) {
		o.rules = append(o.rules, rules...)
	}
}
