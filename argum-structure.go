package argum

import (
	"fmt"
	"reflect"
	"strings"
)

type structure struct {
	i interface{}
	t reflect.Type
	v reflect.Value

	fields []*field

	commandsReq   bool
	commandsTaken bool
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
	args = s.splitShortBooleans(args)
	args = s.splitArgs(args)

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

		if f.command {
			s.commandsTaken = true

			n, err := f.s.parseArgs(args[i+1:])
			i = i + n

			if err != nil && i+1 >= len(args) {
				return i, err
			}

			if f.v.Kind() == reflect.Ptr {
				f.v.Set(reflect.ValueOf(f.s.i))
			} else {
				f.v.Set(reflect.ValueOf(f.s.i).Elem())
			}

			f.taken = true

			continue
		}

		var n int
		switch {
		case f.pos:
			n, err = f.setValue(append([]string{arg}, vals...)...)
			n-- //as is positional argument
		case f.v.Kind() == reflect.Bool:
			n, err = f.setBool(arg, vals)
		default:
			n, err = f.setValue(vals...)
		}

		if err != nil {
			return
		}

		i += n
	}

	for _, f := range s.fields {
		if f.command && !s.commandsTaken {
			return i, fmt.Errorf("Command not selected")
		}
		if !f.taken && f.req {
			return i, fmt.Errorf("Required argument '%s' not set", f.name)
		}
	}

	return
}

func (s *structure) splitArgs(args []string) (newargs []string) {
	for _, arg := range args {

		//append escaped argument
		if len(arg) > 1 && arg[0] == '"' && arg[len(arg)-1] == '"' {
			newargs = append(newargs, trim(arg))
			continue
		}

		//split if contains equal sign
		if len(arg) > 3 && strings.Contains(arg, "=") {
			ss := strings.SplitN(arg, "=", 2)
			arg := ss[0]
			vals := []string{ss[1]}

			f, ok := s.lookupField(arg)
			if ok && f.v.Kind() == reflect.Slice {
				vals = strings.Split(vals[0], ",")
			}

			newargs = append(newargs, arg)
			newargs = append(newargs, vals...)
			continue
		}

		//split short merged key value
		if len(arg) > 2 && matchShort(arg[:2]) {
			if f, ok := s.lookupShortField(arg[:2]); ok {
				vals := []string{arg[2:]}

				if f.v.Kind() == reflect.Slice {
					vals = strings.Split(vals[0], ",")
				}

				newargs = append(newargs, arg[:2])
				newargs = append(newargs, vals...)
				continue
			}
		}

		newargs = append(newargs, trim(arg))
	}

	return
}

func (s *structure) splitArg(arg string) (string, string, bool) {
	if strings.Contains(arg, "=") {
		ss := strings.SplitN(arg, "=", 2)
		return trim(ss[0]), trim(ss[1]), true
	}

	if len(arg) > 2 && matchShort(arg) {
		_, ok := s.lookupShortField(arg[:2])
		if ok {
			return arg[:2], trim(arg[2:]), true
		}
	}

	return trim(arg), "", false
}

func (s *structure) splitShortBooleans(args []string) (newargs []string) {
	for _, arg := range args {
		if len(arg) < 3 || arg[0] != '-' || arg[1] == '-' || arg[2] == '=' {
			newargs = append(newargs, arg)
			continue
		}

		var err error

		subargs := strings.Split(arg[1:], "")
		var boolargs []string

		for _, subarg := range subargs {
			f, ok := s.lookupShortField("-" + subarg)
			if !ok || f.v.Kind() != reflect.Bool {
				err = fmt.Errorf("Unexpected argument %s", arg)
				break
			}
			boolargs = append(boolargs, "-"+subarg)
		}

		if err != nil {
			newargs = append(newargs, arg)
		} else {
			newargs = append(newargs, boolargs...)
		}

	}

	return
}

func (s *structure) getNextValues(osArgs []string) (vals []string) {
	for _, arg := range osArgs {
		var ok bool

		switch {
		case matchLong(arg):
			arg, _, _ := s.splitArg(arg)
			_, ok = s.lookupLongField(arg)
		case matchShort(arg):
			arg, _, _ := s.splitArg(arg)
			_, ok = s.lookupShortField(arg)
		case arg == "--":
			ok = true
		default:
			if f, ok2 := s.lookupField(arg); ok2 && f.command {
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
	for _, f := range s.fields {
		if !f.taken && f.short == arg || f.long == arg {
			return f, true
		}
	}

	for _, f := range s.fields {
		if !f.taken && f.command && f.name == arg && !s.commandsTaken {
			return f, true
		}
	}

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
