package argum

import (
	"fmt"
	"reflect"
	"strings"
)

func prepareStruct(i interface{}) (*userFields, error) {
	uf, t, v := newUserFields(i)

	for i := 0; i < t.NumField(); i++ {
		f, err := uf.newField(t.Field(i), v.Field(i))
		if err != nil {
			return uf, err
		}
		if !f.v.CanSet() {
			continue
		}

		uf.fields = append(uf.fields, f)
	}

	return uf, nil
}

func newUserFields(i interface{}) (uf *userFields, t reflect.Type, v reflect.Value) {
	uf = new(userFields)
	uf.i = i

	t = reflect.TypeOf(i)
	v = reflect.ValueOf(i)
	if v.IsNil() {
		v = reflect.New(t.Elem())
	}

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return
}

func (uf *userFields) newField(rField reflect.StructField, v reflect.Value) (f *field, err error) {
	f = &field{
		field: rField,
		v:     v,
		name:  strings.ToLower(rField.Name),
		help:  rField.Tag.Get("help"),
		def:   rField.Tag.Get("default"),
	}

	if v.CanSet() {
		val, err := f.getDefaultValue()
		if err != nil {
			return f, err
		}
		if val != nil {
			f.v.Set(reflect.ValueOf(val))
		}
	}

	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Struct {
		f.command = true
		f.pos = false

		var ptr interface{}
		switch v.Kind() {
		case reflect.Ptr:
			ptr = reflect.New(v.Type().Elem()).Interface()
			f.commandType = v.Type().Elem().String()
		case reflect.Struct:
			ptr = reflect.New(v.Type()).Interface()
			f.commandType = v.Type().String()
		}

		f.commandFields, err = prepareStruct(ptr)
		return
	}

	t, ok := rField.Tag.Lookup("argum")
	if !ok {
		f.fromName()
		return f, nil
	}

	tags := strings.Split(t, ",")
	for _, tag := range tags {
		if len(tag) <= 1 {
			return f, fmt.Errorf("invalid argument tag `%s`", tag)
		}

		if ok, err := f.structFieldLong(tag); ok {
			if err != nil {
				return f, err
			}
			continue
		}

		if ok, err := f.structFieldShort(tag); ok {
			if err != nil {
				return f, err
			}
			continue
		}

		if tag == "pos" || tag == "positional" {
			f.pos = true
		}

		if tag == "req" || tag == "required" {
			f.req = true
		}

		if strings.Contains(tag, "|") {
			f.opt = strings.Split(tag, "|")
		}
	}

	if f.pos {
		if f.short != "" || f.long != "" {
			return f, fmt.Errorf("invalid `%s`, positional argument can not have long or short keys", rField.Name)
		}
	} else {
		if f.long == "" && f.short == "" {
			f.fromName()
		}
	}

	return
}

func (f *field) fromName() {
	if len(f.name) == 1 {
		f.short = "-" + f.name
	} else {
		f.long = "--" + f.name
	}
}

func (f *field) structFieldLong(tag string) (bool, error) {
	if len(tag) > 2 && tag[0:2] == "--" {
		if len(tag[2:]) > 1 {
			f.long = tag
			return true, nil
		}
		return true, fmt.Errorf("invalid tag `%s` long argument сan not be in 1 character", tag)
	}
	return false, nil
}

func (f *field) structFieldShort(tag string) (bool, error) {
	if tag[0] == '-' {
		if len(tag[1:]) == 1 {
			f.short = tag
			return true, nil
		}
		return true, fmt.Errorf("invalid tag `%s` short argument shoud be only 1 character", tag)
	}
	return false, nil
}
