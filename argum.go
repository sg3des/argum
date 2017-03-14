package argum

import (
	"fmt"
	"net/mail"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	argLong          = regexp.MustCompile("^--[a-zA-Z]{2,}")
	argShort         = regexp.MustCompile("^-[a-z]")
	argShortBooleans = regexp.MustCompile("(?i)^-[a-z]{2,}$")
	argPos           = regexp.MustCompile("(?i)^[a-z0-9].*")
	argValues        = regexp.MustCompile("(?i)^[a-z0-9]+.*")
	boolmatch        = regexp.MustCompile("(true|false)")

	name    string
	Version string
	uf      *userFields
)

type userFields struct {
	i      interface{}
	fields []*field
}

type field struct {
	v     reflect.Value
	field reflect.StructField

	name string

	short string
	long  string
	pos   bool
	req   bool

	taken bool

	help string
	def  string
	opt  []string
}

//MustParse parse os.Args for struct and fatal if it has error
func MustParse(i interface{}) {
	if err := Parse(i); err != nil {
		fmt.Println(err)
		PrintHelp(1)
	}
}

//Parse os.Args for incomimng struct and return error
func Parse(i interface{}) error {
	name = os.Args[0]

	var err error
	uf, err = prepareStruct(i)
	if err != nil {
		return fmt.Errorf("failed prepare structure, reson: %s", err)
	}

	uf.helpRequested(os.Args[1:])
	// uf.funcRequested(os.Args[1:])

	return uf.parseArgs(os.Args[1:])
}

// func (uf *userFields) funcRequested(osArgs []string) {
// 	for i, arg := range osArgs {
// 		if method, ok := uf.lookupMethod(arg); ok {
// 			method.v.Call(nil)
// 			osArgs[i] = ""
// 		}
// 	}
// }

// func (uf *userFields) lookupMethod(name string) (*field, bool) {
// 	for _, f := range uf.fields {
// 		if f.method && (f.name == name || f.long == name) {
// 			f.taken = true
// 			return f, true
// 		}
// 	}
// 	return nil, false
// }

func prepareStruct(i interface{}) (uf *userFields, err error) {
	uf = &userFields{i: i}

	t := reflect.TypeOf(i)
	v := reflect.ValueOf(i)
	if v.IsNil() {
		v = reflect.New(t.Elem())
	}

	// if mv := v.MethodByName("Version"); !mv.IsNil() {
	// 	uf.version = mv
	// }

	// var methods []*field
	// for i := 0; i < v.NumMethod(); i++ {
	// 	name := strings.ToLower(t.Method(i).Name)
	// 	mv := v.Method(i)

	// 	f := &field{name: name, method: true, v: mv}
	// 	if name == "version" {
	// 		f.long = "--version"
	// 		f.help = "display version and exit"
	// 	}
	// 	methods = append(methods, f)
	// }

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	for i := 0; i < t.NumField(); i++ {
		f, err := parseStructFiled(t.Field(i), v.Field(i))
		if err != nil {
			return uf, err
		}

		uf.fields = append(uf.fields, f)
	}

	// uf.fields = append(uf.fields, methods...)

	return
}

func (uf userFields) parseArgs(osArgs []string) (err error) {
	osArgs = splitShortBooleans(osArgs)

	for i := 0; i < len(osArgs); i++ {
		if osArgs[i] == "" {
			continue
		}

		argname, val := splitArg(osArgs[i])
		vals := getNextValues(osArgs, i)

		var f *field
		var ok bool

		switch {
		case argLong.MatchString(argname):
			f, ok = uf.lookupArgByLong(argname)
		case argShort.MatchString(argname):
			f, ok = uf.lookupArgByShort(argname)
		default:
			f, ok = uf.lookupArgByPos()
		}

		if ok {

			// if a.f.Name != "" {
			// 	a.f.Func.Call([]reflect.Value{})
			// } else {
			f.taken = true
			err = f.setArgument(argname, val, vals, &i)
			// }
		} else {
			err = fmt.Errorf("Unexpected argument `%s`", argname)
		}

		if err != nil {
			return err
		}
	}

	for _, f := range uf.fields {
		if f.req && !f.taken {
			return fmt.Errorf("required argument `%s` not set", f.name)
		}
	}

	return nil
}

//splitShortBooleans split one args multiple booleans, ex: "-abc"
func splitShortBooleans(osArgs []string) []string {
	var newOsArgs []string

	for _, arg := range osArgs {

		if len(arg) > 2 && argShort.MatchString(arg) {

			var newArgs []string
			var done bool
			for _, s := range strings.Split(arg[1:], "") {

				f, ok := uf.lookupArgByShort("-" + s)
				if !ok {
					done = false
					break
				}
				if f.v.Kind() != reflect.Bool {
					done = false
					break
				}
				done = true
				newArgs = append(newArgs, "-"+s)
			}
			if done {
				newOsArgs = append(newOsArgs, newArgs...)
				continue
			}
		}

		newOsArgs = append(newOsArgs, arg)
	}

	return newOsArgs
}

func (uf *userFields) lookupArgByLong(argname string) (*field, bool) {
	for _, f := range uf.fields {
		if f.long == argname && !f.taken {
			// f.taken = true
			return f, true
		}
	}
	return nil, false
}

func (uf *userFields) lookupArgByShort(argname string) (*field, bool) {
	for _, f := range uf.fields {
		if f.short == argname && !f.taken {
			// f.taken = true
			return f, true
		}
	}
	return nil, false
}

func (uf *userFields) lookupArgByPos() (*field, bool) {
	for _, f := range uf.fields {
		if f.pos && !f.taken {
			// f.taken = true
			return f, true
		}
	}
	return nil, false
}

func (f *field) setArgument(argname, val string, vals []string, i *int) (err error) {
	// log.Println(a.name, a.pos, argname, val, vals, *i)
	if f.v.Kind() == reflect.Bool {
		valbool := true

		if boolmatch.MatchString(val) {
			valbool, _ = strconv.ParseBool(val)
		} else if len(vals) > 0 && boolmatch.MatchString(vals[0]) {
			*i++
			valbool, _ = strconv.ParseBool(vals[0])
		}

		f.v.SetBool(valbool)
		return
	}

	if f.pos {
		if f.v.Kind() == reflect.Slice {
			*i = *i + len(vals)
			return f.setSlice(append([]string{argname}, vals...))
		} else {
			return f.setValue(argname)
		}
	}

	if f.v.Kind() == reflect.Slice {
		if val != "" {
			return f.setSlice(strings.Split(val, ","))
		}

		*i = *i + len(vals)
		return f.setSlice(vals)
	} else {
		if val == "" && len(vals) > 0 {
			val = vals[0]
			*i++
		}
		return f.setValue(val)
	}

	return nil
}

func (f *field) getDefaultValue() (interface{}, error) {
	if !f.v.CanSet() {
		return nil, nil
	}

	if f.def == "" {
		return nil, nil
	}

	slice := strings.Split(f.def, ",")

	switch f.v.Interface().(type) {
	case string:
		return f.def, nil
	case bool:
		return strconv.ParseBool(f.def)
	case int:
		return strconv.Atoi(f.def)
	case float32:
		f, err := strconv.ParseFloat(f.def, 32)
		return float32(f), err
	case float64:
		return strconv.ParseFloat(f.def, 64)
	case time.Duration:
		return time.ParseDuration(f.def)
	case *mail.Address:
		return mail.ParseAddress(f.def)

	case []string:
		return slice, nil

	case []bool:
		var vals []bool
		for _, s := range slice {
			val, err := strconv.ParseBool(s)
			if err != nil {
				return nil, err
			}
			vals = append(vals, val)
		}
		return vals, nil

	case []int:
		var vals []int
		for _, s := range slice {
			val, err := strconv.Atoi(s)
			if err != nil {
				return nil, err
			}
			vals = append(vals, val)
		}
		return vals, nil

	case []float32:
		var vals []float32
		for _, s := range slice {
			val, err := strconv.ParseFloat(s, 32)
			if err != nil {
				return nil, err
			}
			vals = append(vals, float32(val))
		}
		return vals, nil

	case []float64:
		var vals []float64
		for _, s := range slice {
			val, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return nil, err
			}
			vals = append(vals, val)
		}
		return vals, nil

	case []time.Duration:
		var vals []time.Duration
		for _, s := range slice {
			val, err := time.ParseDuration(s)
			if err != nil {
				return nil, err
			}
			vals = append(vals, val)
		}
		return vals, nil

	case []*mail.Address:
		var vals []*mail.Address
		for _, s := range slice {
			val, err := mail.ParseAddress(s)
			if err != nil {
				return nil, err
			}
			vals = append(vals, val)
		}
		return vals, nil

	}
	return nil, nil
}

func (f *field) setValue(val string) error {
	if len(f.opt) > 0 && !f.checkOpt(val) {
		return fmt.Errorf("impossible value %s, you should choose from %v", val, f.opt)
	}

	switch f.v.Interface().(type) {

	case string:
		f.v.SetString(val)

	case int:
		i, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		f.v.SetInt(int64(i))

	case float32:
		float, err := strconv.ParseFloat(val, 32)
		if err != nil {
			return err
		}
		f.v.SetFloat(float)

	case float64:
		float, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return err
		}
		f.v.SetFloat(float)

	case time.Duration:
		d, err := time.ParseDuration(val)
		if err != nil {
			return err
		}
		f.v.Set(reflect.ValueOf(d))

	case *mail.Address:
		m, err := mail.ParseAddress(val)
		if err != nil {
			return err
		}
		f.v.Set(reflect.ValueOf(m))
	}

	return nil
}

func (f *field) checkOpt(val string) bool {
	for _, o := range f.opt {
		if o == val {
			return true
		}
	}
	return false
}

func (f *field) setSlice(vals []string) error {
	if len(f.opt) > 0 && !f.checkOptSlice(vals) {
		return fmt.Errorf("impossible values %v, you should choose from %v", vals, f.opt)
	}

	switch f.v.Interface().(type) {

	case []string:
		f.v.Set(reflect.ValueOf(vals))

	case []int:
		var ints []int
		for _, s := range vals {
			i, err := strconv.Atoi(s)
			if err != nil {
				return err
			}
			ints = append(ints, i)
		}
		f.v.Set(reflect.ValueOf(ints))

	case []float32:
		var floats []float32
		for _, s := range vals {
			f, err := strconv.ParseFloat(s, 32)
			if err != nil {
				return err
			}
			floats = append(floats, float32(f))
		}
		f.v.Set(reflect.ValueOf(floats))

	case []float64:
		var floats []float64
		for _, s := range vals {
			f, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return err
			}
			floats = append(floats, f)
		}
		f.v.Set(reflect.ValueOf(floats))

	case []time.Duration:
		var durs []time.Duration
		for _, s := range vals {
			d, err := time.ParseDuration(s)
			if err != nil {
				return err
			}
			durs = append(durs, d)
		}
		f.v.Set(reflect.ValueOf(durs))

	case []*mail.Address:
		var maddrs []*mail.Address
		for _, s := range vals {
			m, err := mail.ParseAddress(s)
			if err != nil {
				return err
			}
			maddrs = append(maddrs, m)
		}
		f.v.Set(reflect.ValueOf(maddrs))
	}

	return nil
}

func (f *field) checkOptSlice(vals []string) bool {
	var present int
	for _, o := range f.opt {
		if o == "" {
			continue
		}

		for _, v := range vals {
			if o == v {
				present++
				o = ""
			}
		}
	}

	if present == len(vals) {
		return true
	}
	return false
}

func splitArg(s string) (argname string, value string) {
	// log.Println(s)
	var argVal []string
	if strings.Contains(s, "=") {
		argVal = strings.SplitN(s, "=", 2)
		return argVal[0], strings.Trim(argVal[1], "\"")
	}

	if len(s) > 2 && argShort.MatchString(s) {
		f, ok := uf.lookupArgByShort(s[:2])
		if ok && f.v.Kind() != reflect.String {
			// log.Println(s[:2], s[2:])
			return s[:2], s[2:]
		}
	}

	return strings.Trim(s, "\""), ""
}

func getNextValues(osArgs []string, i int) (vals []string) {
	i++
	for ; i < len(osArgs); i++ {
		s := strings.Trim(osArgs[i], "\"")

		var ok bool

		switch {
		case argLong.MatchString(s):
			_, ok = uf.lookupArgByLong(s)
		case argShort.MatchString(s):
			_, ok = uf.lookupArgByShort(s)
		}

		if ok {
			return
		}
		// if !argValues.MatchString(s) {
		// 	return
		// }
		vals = append(vals, s)
	}

	return
}

func parseStructFiled(rField reflect.StructField, v reflect.Value) (*field, error) {
	f := &field{
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

	return f, nil
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
