package schema

import (
	"reflect"
	"strings"
	"sync"

	"github.com/ygrebnov/errorc"

	"github.com/ygrebnov/model/pkg/errors"
	"github.com/ygrebnov/model/pkg/keys"
)

const (
	tagJSON         = "json"
	tagYAML         = "yaml"
	tagENV          = "env"
	tagValidate     = "validate"
	tagValidateElem = "validateElem"
	tagDefault      = "default"
	tagDefaultElem  = "defaultElem"
)

type N struct {
	Name            []string // all segments (ordered) starting from the root
	T               reflect.Type
	I               []int // reflect.StructField.Index
	JSONTag         string
	YAMLTag         string
	Env             []string // all env tags (ordered) starting from the root
	DefaultTag      string
	DefaultElemTag  string
	ValidateTag     string
	ValidateElemTag string
	Children        []*N
}

func (n *N) GetName(separator string) string {
	return strings.Join(n.Name, separator)
}

type Controller struct {
	mu sync.RWMutex

	Tree  *N
	Index map[string]*N // N.fullName (concatenated N.Name) -> *N
}

func (c *Controller) Add(name string, node *N) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Index[name] = node
}

func NewN[T any](c *Controller) (*N, error) {
	var zero *T // never dereferenced

	v := reflect.ValueOf(zero).Elem()
	if v.Kind() != reflect.Struct {
		return nil, errors.ErrTypeParamNotStruct
	}

	n := &N{}
	if err := parse(v, n, c); err != nil {
		return nil, errorc.With(
			errors.ErrCannotCompileSchema,
			errorc.String(keys.ObjectType, v.Type().String()),
			errorc.Error(keys.Cause, err),
		)
	}

	return n, nil
}

func parse(v reflect.Value, n *N, c *Controller) error {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		newN := &N{
			Name:            append(n.Name, strings.ToLower(field.Name)),
			T:               field.Type,
			I:               field.Index,
			JSONTag:         field.Tag.Get(tagJSON),
			YAMLTag:         field.Tag.Get(tagYAML),
			Env:             append(n.Env, field.Tag.Get(tagENV)),
			DefaultTag:      field.Tag.Get(tagDefault),
			DefaultElemTag:  field.Tag.Get(tagDefaultElem),
			ValidateTag:     field.Tag.Get(tagValidate),
			ValidateElemTag: field.Tag.Get(tagValidateElem),
		}

		n.Children = append(n.Children, newN)
		c.Add(newN.GetName("."), newN)

		switch field.Type.Kind() {
		case reflect.Struct:
			// recurse into struct fields
			if err := parse(v.Field(i), newN, c); err != nil {
				return err
			}
		case reflect.Ptr:
			if field.Type.Elem().Kind() == reflect.Struct {
				// recurse into pointer to struct fields
				if err := parse(v.Field(i).Elem(), newN, c); err != nil {
					return err
				}
			}
		case reflect.Slice:
		// TODO
		case reflect.Map:
		// TODO
		case reflect.Interface:
			// TODO
		}
	}
	return nil
}
