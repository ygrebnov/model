// Package field defines the external value source and sink contracts used by
// model bindings.
package field

// ValueSource supplies typed values to Binding.ApplyValues.
type ValueSource interface {
	// Get returns the value for name, whether a value was found, and any lookup
	// error. Names are exact exported schema paths such as "Name" and
	// "Server.Host"; collection fields use a "[]" suffix such as "Items[]".
	// A found nil value resets the target field to its zero value.
	Get(name string) (any, bool, error)
}

// EnvSource looks up a raw environment value by name.
//
// Bindings use an EnvSource internally to snapshot process environment values
// during construction.
type EnvSource interface {
	// Lookup returns the value for name and whether it exists.
	Lookup(name string) (value string, found bool)
}

// ValueSink receives values exported by Binding.WriteValues.
type ValueSink interface {
	// Set receives a schema path and its current field value. Names are exact
	// exported schema paths such as "Name" and "Server.Host". Collections of
	// structs use a "[]" suffix such as "Items[]"; scalar collections use their
	// field path, such as "Tags". Child paths can repeat for multiple collection
	// elements. Reference values are passed without deep copying.
	Set(name string, value any) error
}
