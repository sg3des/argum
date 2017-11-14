package argum

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
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
			return i, fmt.Errorf("unexpected argument '%s'", args[i])
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
			return i, fmt.Errorf("required argument '%s' not set", f.name)
		}
	}

	return
}

func (s *structure) prepareArgs(osArgs []string) (newArgs []string, err error) {
	for _, arg := range osArgs {

		switch {
		case matchEscape(arg):
			newArgs = append(newArgs, trim(arg))
		case strings.Contains(arg, "="):
			ss := strings.SplitN(arg, "=", 2)

			key := ss[0]
			if ok := s.recursiveArgExists(key); !ok {
				return newArgs, fmt.Errorf("unexpected argument '%s'", key)
			}

			vals := splitArgs(ss[1])

			newArgs = append(newArgs, key)
			newArgs = append(newArgs, vals...)
		case matchShort(arg) && len(arg) > 2:
			key := arg[:2]
			if ok := s.recursiveArgExists(key); !ok {
				return newArgs, fmt.Errorf("unexpected argument '%s'", key)
			}
			keys, err := s.splitShortArgs(arg[2:])
			if err != nil {
				return newArgs, err
			}

			newArgs = append(newArgs, key)
			newArgs = append(newArgs, keys...)
		default:
			newArgs = append(newArgs, arg)
		}

	}

	return newArgs, nil
}

func (s *structure) recursiveArgExists(arg string) bool {
	for _, f := range s.fields {
		switch {
		case f.long == arg:
			return true
		case f.short == arg:
			return true
		case f.cmd && f.name == arg:
			return true
		}
	}

	for _, f := range s.fields {
		if f.s != nil && len(f.s.fields) > 0 {
			ok := f.s.recursiveArgExists(arg)
			if ok {
				return ok
			}
		}
	}

	return false
}

func (s *structure) splitShortArgs(arg string) ([]string, error) {
	if _, err := strconv.Atoi(arg); err == nil {
		return []string{arg}, nil
	}

	if _, err := strconv.ParseFloat(arg, 64); err == nil {
		return []string{arg}, nil
	}

	if _, err := time.ParseDuration(arg); err == nil {
		return []string{arg}, nil
	}

	if _, err := strconv.ParseBool(arg); err == nil {
		return []string{arg}, nil
	}

	var args []string
	for _, b := range arg {
		short := "-" + string(b)
		if ok := s.recursiveArgExists(short); !ok {
			return args, fmt.Errorf("failed parse short defined argument '%s'", arg)
		}

		args = append(args, short)
	}

	return args, nil
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
