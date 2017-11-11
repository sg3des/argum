package argum

import (
	"fmt"
	"reflect"
)

type structure struct {
	i interface{}
	t reflect.Type
	v reflect.Value

	fields []*field
	sel    bool
	taken  bool
}

func prepareStructure(i interface{}) (*structure, error) {
	s := newStructure(i)

	//prepare fields
	for i := 0; i < s.t.NumField(); i++ {
		v := s.v.Field(i)
		if !v.CanSet() {
			continue
		}

		f, err := s.newField(s.t.Field(i), v)
		if err != nil {
			return s, err
		}

		s.fields = append(s.fields, f)
	}

	return s, nil
}

func newStructure(i interface{}) *structure {
	s := new(structure)
	s.i = i

	s.t = reflect.TypeOf(i)
	s.v = reflect.ValueOf(i)
	if s.v.IsNil() {
		s.v = reflect.New(s.t.Elem())
	}

	if s.v.Kind() == reflect.Ptr {
		s.v = s.v.Elem()
	}
	if s.t.Kind() == reflect.Ptr {
		s.t = s.t.Elem()
	}

	return s
}

func (s *structure) parseArgs(args []string) (i int, err error) {

	for i = 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			continue
		}

		vals := s.getNextValues(args[i+1:])

		f, ok := s.lookupField(arg)
		if !ok {
			return i, fmt.Errorf("Unexpected argument '%s'", args[i])
		}

		var n int

		switch {
		case f.sel:
			n, err = f.setStruct(args[i:])
		case f.cmd:
			n, err = f.setStruct(args[i+1:])
		case f.pos:
			n, err = f.setValue(append([]string{arg}, vals...)...)
			n-- //as is positional argument
		case f.v.Kind() == reflect.Bool:
			n, err = f.setBool(arg, vals)
		default:
			n, err = f.setValue(vals...)
		}

		i += n

		if (f.sel || f.cmd) && err != nil && i+1 < len(args) {
			err = nil
		}

		if err != nil || s.sel {
			return
		}
	}

	for _, f := range s.fields {
		if !f.taken && f.req {
			return i, fmt.Errorf("Required argument '%s' not set", f.name)
		}
	}

	return
}

func (s *structure) getNextValues(osArgs []string) (vals []string) {
	for _, arg := range osArgs {
		var ok bool

		switch {
		case matchLong(arg):
			_, ok = s.lookupLongField(arg)
		case matchShort(arg):
			_, ok = s.lookupShortField(arg)
		case arg == "--":
			ok = true
		default:
			if f, ok2 := s.lookupField(arg); ok2 && f.cmd {
				ok = true
			}
		}

		if ok {
			return
		}

		vals = append(vals, trim(arg))
	}
	return
}

func (s *structure) lookupField(arg string) (*field, bool) {

	//selections
	for _, f := range s.fields {
		if !f.taken && f.sel {
			return f, true
		}
	}

	//short and log options
	for _, f := range s.fields {
		if !f.taken && (f.short == arg || f.long == arg) {
			return f, true
		}
	}

	//commands
	for _, f := range s.fields {
		if !f.taken && f.cmd && f.name == arg {
			return f, true
		}
	}

	//positionals
	for _, f := range s.fields {
		if !f.taken && f.pos {
			return f, true
		}
	}

	return nil, false
}

func (s *structure) lookupLongField(arg string) (*field, bool) {
	for _, f := range s.fields {
		if f.long == arg && !f.taken {
			return f, true
		}
	}
	return nil, false
}

func (s *structure) lookupShortField(arg string) (*field, bool) {
	for _, f := range s.fields {
		if f.short == arg && !f.taken {
			return f, true
		}
	}
	return nil, false
}
