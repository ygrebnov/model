package core

import (
	"reflect"

	"github.com/ygrebnov/model/internal/schema"
)

func (s *Service) schemaFor(t reflect.Type) (*schema.Schema, error) {
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if cached, ok := s.schemas.Load(t); ok {
		return cached.(*schema.Schema), nil
	}

	compiled, err := schema.Compile(t)
	if err != nil {
		return nil, err
	}

	actual, _ := s.schemas.LoadOrStore(t, compiled)
	return actual.(*schema.Schema), nil
}
