package field

/*
type Field struct {
	Path            string
	Name            string
	Type            reflect.Type
	JSONName        string
	EnvPath         []string
	DefaultTag      string
	DefaultElemTag  string
	ValidateTag     string
	ValidateElemTag string
}
*/

type ValueSource interface {
	Get(name string) (any, bool, error)
}

type EnvSource interface {
	Lookup(name string) (value string, found bool)
}

type ValueSink interface {
	Set(name string, value any) error
}
