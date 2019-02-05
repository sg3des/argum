package argum

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type field struct {
	v     reflect.Value
	field reflect.StructField

	name string

	short        string
	long         string
	shortboolean bool
	pos          bool
	req          bool
	cmd          bool
	oneof        bool
	emb          bool
	variants     []string

	help string
	def  string

	taken bool
	s     *structure
}

func (s *structure) newField(sf reflect.StructField, v reflect.Value) (f *field, err error) {
	f = &field{
		v:     v,
		field: sf,
		name:  strings.ToLower(sf.Name),
		help:  sf.Tag.Get("help"),
		def:   sf.Tag.Get("default"),
	}

	//prepare commands
	if f.v.Kind() == reflect.Ptr || f.v.Kind() == reflect.Struct {
		f.cmd = true

		var ptr interface{}

		switch v.Kind() {
		case reflect.Ptr:
			ptr = reflect.New(v.Type().Elem()).Interface()
		case reflect.Struct:
			ptr = reflect.New(v.Type()).Interface()
		}

		f.s, err = prepareStructure(ptr)
		if err != nil {
			return
		}
	}

	//set default values
	if !f.cmd && f.def != "" && v.CanSet() {
		if v.Interface() == reflect.Zero(v.Type()).Interface() {
			x, err := f.transformValue(strings.Split(f.def, ","))
			if err != nil {
				return f, err
			}
			f.v.Set(x)
		}
	}

	tag, ok := sf.Tag.Lookup("argum")
	if !ok {
		f.autoShortLong(f.name)
		return
	}

	for _, key := range strings.Split(tag, ",") {
		switch {
		case matchShort(key):
			f.short = key
		case matchLong(key):
			f.long = key
		case key == "pos" || key == "positional":
			f.pos = true
		case key == "req" || key == "required":
			f.req = true
		case key == "oneof":
			f.oneof = true
			f.s.oneof = true
		case strings.Contains(key, "|"):
			f.variants = strings.Split(key, "|")
		case key == "emb" || key == "embedded":
			f.emb = true
			f.s.emb = true
		default:
			err = fmt.Errorf("argument '%s' have unexpected tag description: %s", f.name, key)
		}
	}

	if f.pos {
		if f.short != "" || f.long != "" {
			err = fmt.Errorf("invalid `%s`, positional argument can not have long or short keys", f.name)
			return
		}
	} else {
		if f.short == "" && f.long == "" {
			f.autoShortLong(f.name)
		}
	}

	if f.short != "" && f.v.Kind() == reflect.Bool && !f.req {
		f.shortboolean = true
	}

	method := strings.Title(f.name) + "Variants"
	if m := s.v.MethodByName(method); m.IsValid() {
		ss := m.Call([]reflect.Value{})
		if len(ss) == 1 {
			f.variants = ss[0].Interface().([]string)
		}
	}

	return
}

func (f *field) autoShortLong(fieldname string) {
	fieldname = strings.ToLower(fieldname)
	if len(fieldname) == 1 {
		f.short = "-" + fieldname
	} else {
		f.long = "--" + fieldname
	}
}

func (f *field) nameMatch(arg string) bool {
	if matchLong(arg) {
		return f.long == arg
	}
	if matchShort(arg) {
		return f.short == arg
	}
	return f.name == arg
}

func (f *field) setBool(arg string, vals []string, next []string) (int, error) {

	if len(next) > 0 {
		_, err := strconv.ParseBool(next[0])
		if err == nil {
			_, err = f.setValue(next[0])
			return 1, err
		}
	}

	if len(vals) > 0 {
		_, err := strconv.ParseBool(vals[0])
		if err == nil {
			_, err = f.setValue(vals[0])
			return 0, err
		}
	}

	if f.nameMatch(arg) {
		_, err := f.setValue("true")
		if err != nil {
			return 0, err
		}
		return 0, nil
	}

	return 0, fmt.Errorf("unexpected value %s", arg)
}

func (f *field) setStruct(args []string) (int, error) {
	n, err := f.s.parseArgs(args)

	f.taken = true

	if f.v.Kind() == reflect.Ptr {
		f.v.Set(reflect.ValueOf(f.s.i))
	} else {
		f.v.Set(reflect.ValueOf(f.s.i).Elem())
	}

	if f.oneof {
		for _, f := range f.s.fields {
			f.taken = true
		}
	}

	return n, err
}

func (f *field) setValue(vals ...string) (int, error) {
	if len(vals) == 0 {
		return 0, fmt.Errorf("for field `%s` value is not set", f.name)
	}

	if len(f.variants) > 0 {
		if !contains(f.variants, vals[0]) {
			return 0, fmt.Errorf("impossible value %s, choose from %s", vals[0], f.variants)
		}
	}

	rv, err := f.transformValue(vals)
	if err != nil {
		return 0, err
	}

	f.taken = true
	f.v.Set(rv)

	if rv.Kind() == reflect.Slice {
		return rv.Len(), nil
	}

	return 1, nil
}

func (f *field) transformValue(vals []string) (rv reflect.Value, err error) {
	var x interface{}

	switch f.v.Interface().(type) {
	case bool:
		x, err = strconv.ParseBool(vals[0])
	case string:
		x = vals[0]
	case int:
		x, err = strconv.Atoi(vals[0])
	case float32:
		var _x float64
		_x, err = strconv.ParseFloat(vals[0], 32)
		x = float32(_x)
	case float64:
		x, err = strconv.ParseFloat(vals[0], 64)
	case time.Duration:
		x, err = time.ParseDuration(vals[0])
	case []string:
		x = vals
	case []bool:
		x, err = sliceToBool(vals)
	case []int:
		x, err = sliceToInt(vals)
	case []float32:
		x, err = sliceToFloat32(vals)
	case []float64:
		x, err = sliceToFloat64(vals)
	case []time.Duration:
		x, err = sliceToTimeDuration(vals)
	default:
		err = fmt.Errorf("field %s has unsupported type %T", f.field.Name, f.v.Interface())
	}

	return reflect.ValueOf(x), err
}

func sliceToBool(ss []string) (bools []bool, err error) {
	var b bool
	for _, s := range ss {
		b, err = strconv.ParseBool(s)
		if err != nil {
			if len(bools) > 0 {
				err = nil
			}
			return
		}
		bools = append(bools, b)
	}

	return
}

func sliceToInt(ss []string) (ints []int, err error) {
	var i int
	for _, s := range ss {
		i, err = strconv.Atoi(s)
		if err != nil {
			if len(ints) > 0 {
				err = nil
			}
			return
		}
		ints = append(ints, i)
	}
	return
}

func sliceToFloat32(ss []string) (ints []float32, err error) {
	var i float64
	for _, s := range ss {
		i, err = strconv.ParseFloat(s, 32)
		if err != nil {
			if len(ints) > 0 {
				err = nil
			}
			return
		}
		ints = append(ints, float32(i))
	}
	return
}

func sliceToFloat64(ss []string) (ints []float64, err error) {
	var i float64
	for _, s := range ss {
		i, err = strconv.ParseFloat(s, 64)
		if err != nil {
			if len(ints) > 0 {
				err = nil
			}
			return
		}
		ints = append(ints, i)
	}
	return
}

func sliceToTimeDuration(ss []string) (td []time.Duration, err error) {
	var d time.Duration
	for _, s := range ss {
		d, err = time.ParseDuration(s)
		if err != nil {
			if len(td) > 0 {
				err = nil
			}
			return
		}
		td = append(td, d)
	}

	return
}
