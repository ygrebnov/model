package model

type options struct {
	envPrefix string
	rules     []Rule
}

// Option customizes Binding construction and one-time wrapper operations.
type Option func(o *options)

// WithEnvPrefix adds a non-empty environment variable name prefix for ApplyEnv
// and ValidateWithDefaults. For example, prefix "MYAPP" makes an untagged
// Name field use MYAPP_NAME.
func WithEnvPrefix(prefix string) Option {
	return func(o *options) {
		if prefix != "" {
			o.envPrefix = prefix
		}
	}
}

// WithRules registers custom validation rules during binding construction.
//
// Repeated WithRules options compose in declaration order. See Rule and NewRule
// for details on creating rules.
func WithRules(rules ...Rule) Option {
	return func(o *options) {
		o.rules = append(o.rules, rules...)
	}
}
