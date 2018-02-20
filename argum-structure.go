package argum

import (
	"fmt"
	"log"
	"reflect"
	"strings"
)

type structure struct {
	i interface{}
	t reflect.Type
	v reflect.Value

	fields []*field
	oneof  bool
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

	s.v = reflect.ValueOf(i)
	if s.v.Kind() != reflect.Ptr {
		log.Panicf("%s is not a pointer on struct", s.v.Type())
	}
	s.v = s.v.Elem()
	s.t = s.v.Type()

	if s.v.Kind() == reflect.Ptr && s.v.IsNil() {
		s.t = s.t.Elem()

		s.v.Set(reflect.New(s.t))
		s.v = s.v.Elem()
	}

	return s
}

func (s *structure) parseArgs(args []string) (i int, err error) {

	for i = 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			continue
		}

		if matchSortBooleans(arg) {
			shortargs, err := s.splitShortBooleans(arg)
			if err != nil {
				return i, err
			}
			if _, err = s.parseArgs(shortargs); err != nil {
				return i, err
			}
			continue
		}

		arg, vals := splitArg(arg)

		f, ok := s.lookupField(arg)
		if !ok {
			return i, fmt.Errorf("unexpected argument '%s'", args[i])
		}

		var n int
		var x int
		var next []string

		if len(vals) == 0 && i+1 < len(args) {
			next, x = s.getNextValues(args[i+1:])
		}

		switch {
		case f.oneof:
			n, err = f.setStruct(args[i:])
		case f.cmd:
			n, err = f.setStruct(args[i+1:])
		case f.v.Kind() == reflect.Bool:
			n, err = f.setBool(arg, vals, next)
		case f.pos:
			if len(vals) > 0 {
				_, err = f.setValue(append([]string{arg}, vals...)...)
			} else {
				n, err = f.setValue(append([]string{arg}, next...)...)
				if n > 1 {
					n = x
				} else {
					n = 0
				}
			}
		default:
			if len(vals) > 0 {
				_, err = f.setValue(vals...)
			} else {
				n, err = f.setValue(next...)
				if n > x {
					n = x
				}
			}
		}

		i += n

		if (f.oneof || f.cmd) && err != nil && i+1 < len(args) {
			err = nil
		}

		if err != nil || s.oneof {
			return
		}
	}

	for _, f := range s.fields {
		if f.req && !f.taken {
			return i, fmt.Errorf("required argument '%s' not set", f.name)
		}
	}

	return
}

func (s *structure) splitShortBooleans(arg string) (shorts []string, err error) {
	for _, b := range arg[1:] {
		short := "-" + string(b)
		if !s.recShortBoolExists(short) {
			err = fmt.Errorf("failed parse short defined arguments '%s'", arg)
			return
		}
		shorts = append(shorts, short)
	}

	return
}

func (s *structure) recShortBoolExists(arg string) bool {
	for _, f := range s.fields {
		if f.short == arg && f.v.Kind() == reflect.Bool {
			return true
		}
	}

	for _, f := range s.fields {
		if f.s != nil && len(f.s.fields) > 0 {
			if ok := f.s.recShortBoolExists(arg); ok {
				return true
			}
		}
	}

	return false
}

func (s *structure) recursiveArgExists(arg string) (*field, bool) {
	for _, f := range s.fields {
		switch {
		case f.long == arg:
			return f, true
		case f.short == arg:
			return f, true
		case f.cmd && f.name == arg:
			return f, true
		}
	}

	for _, f := range s.fields {
		if f.s != nil && len(f.s.fields) > 0 {
			f, ok := f.s.recursiveArgExists(arg)
			if ok {
				return f, ok
			}
		}
	}

	return nil, false
}

func (s *structure) getNextValues(osArgs []string) (vals []string, n int) {

	for i, arg := range osArgs {
		var ok bool

		switch {
		case matchLong(arg):
			ok = true
			// _, ok = s.lookupLongField(arg)
		case matchShort(arg):
			ok = true
			// _, ok = s.lookupShortField(arg)
		case arg == "--":
			ok = true
		case strings.Contains(arg, ",") && i == 0:
			vals = append(vals, splitValues(arg)...)
			n = 1
			ok = true
		default:
			if f, ok2 := s.lookupField(arg); ok2 && f.cmd {
				ok = true
			}
		}

		if ok {
			return
		}

		if matchEscape(arg) {
			arg = trim(arg)
		}

		n++
		vals = append(vals, arg)
	}
	return
}

func (s *structure) lookupField(arg string) (*field, bool) {

	//selections
	for _, f := range s.fields {
		if !f.taken && f.oneof {
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
