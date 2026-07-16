package field

type ValueSource interface {
	Get(name string) (any, bool, error)
}

type EnvSource interface {
	Lookup(name string) (value string, found bool)
}

type ValueSink interface {
	Set(name string, value any) error
}
