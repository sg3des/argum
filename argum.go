package argum

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

	mode       bool
	modeType   string
	modeFields *userFields
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

	if filepath.Ext(name) == ".test" {
		return nil
	}

	var err error
	uf, err = prepareStruct(i)
	if err != nil {
		return fmt.Errorf("failed prepare structure, reson: %s", err)
	}

	uf.helpRequested(os.Args[1:])
	// uf.funcRequested(os.Args[1:])

	_, err = uf.parseArgs(os.Args[1:])
	return err
}

func (uf *userFields) parseArgs(osArgs []string) (i int, err error) {
	osArgs = splitShortBooleans(osArgs)

	for i = 0; i < len(osArgs); i++ {
		if osArgs[i] == "" {
			continue
		}

		if osArgs[i] == "--" {
			continue
		}

		argname, val := splitArg(osArgs[i])
		vals := getNextValues(osArgs, i)

		var f *field
		var err error

		switch {
		case argLong.MatchString(argname):
			f, err = uf.lookupArgByLong(argname)
		case argShort.MatchString(argname):
			f, err = uf.lookupArgByShort(argname)
		default:
			argname = trim(osArgs[i])

			f, err = uf.lookupArgByMode(argname)
			if err != nil {
				f, err = uf.lookupArgByPos(argname)
			}
		}
		if err != nil {
			return i, err
		}

		if f.mode {
			f.taken = true
			i++
			ni, err := f.modeFields.parseArgs(osArgs[i:])
			if err != nil && ni+i == len(osArgs) {
				return ni + i, err
			}
			if f.v.Kind() == reflect.Ptr {
				f.v.Set(reflect.ValueOf(f.modeFields.i))
			} else {
				f.v.Set(reflect.ValueOf(f.modeFields.i).Elem())
			}

			continue
		}

		f.taken = true
		err = f.setArgument(argname, val, vals, &i)
		if err != nil {
			return i, err
		}
	}

	for _, f := range uf.fields {
		if f.req && !f.taken {
			return i, fmt.Errorf("required argument `%s` not set", f.name)
		}
	}

	return i, nil
}

//splitShortBooleans split one args multiple booleans, ex: "-abc"
func splitShortBooleans(osArgs []string) []string {
	var newOsArgs []string

	for _, arg := range osArgs {

		if len(arg) > 2 && argShort.MatchString(arg) {

			var newArgs []string
			var done bool
			for _, s := range strings.Split(arg[1:], "") {

				f, err := uf.lookupArgByShort("-" + s)
				if err != nil {
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

func (uf *userFields) lookupArgByLong(argname string) (*field, error) {
	for _, f := range uf.fields {
		if f.long == argname && !f.taken {
			return f, nil
		}
	}
	return nil, fmt.Errorf("Unexpected argument `%s`", argname)
}

func (uf *userFields) lookupArgByShort(argname string) (*field, error) {
	for _, f := range uf.fields {
		if f.short == argname && !f.taken {
			return f, nil
		}
	}
	return nil, fmt.Errorf("Unexpected argument `%s`", argname)
}

func (uf *userFields) lookupArgByPos(argname string) (*field, error) {
	for _, f := range uf.fields {
		if f.pos && !f.taken {
			return f, nil
		}
	}
	return nil, fmt.Errorf("Unexpected positional argument `%s`", argname)
}

func (uf *userFields) lookupArgByMode(argname string) (f *field, err error) {
	for _, f = range uf.fields {
		if !f.mode {
			continue
		}

		//Call check method
		if ss, ok := f.callMethod("Check", argname); ok {
			if err := ss.Interface(); err != nil {
				return f, fmt.Errorf("%s", err)
			}
			return f, nil
		}

		//Call Variants method
		if ss, ok := f.callMethod("Variants"); ok {
			variants := ss.Interface().([]string)
			if !contains(variants, argname) {
				return f, fmt.Errorf("Unexpected argument `%s`", argname)
			}

			return f, nil
		}

		if f.name == argname && !f.taken {
			break
		}
	}

	if !f.mode {
		return nil, fmt.Errorf("Unexpected mode `%s`", argname)
	}

	for _, sf := range uf.fields {
		if !sf.mode {
			continue
		}
		if f.name != sf.name && sf.modeType == f.modeType && sf.taken {
			return nil, fmt.Errorf("mode is already set")
		}
	}

	return
}

func (f *field) callMethod(name string, args ...interface{}) (reflect.Value, bool) {
	// log.Println(name)
	method := f.v.MethodByName(name)
	if !method.IsValid() {
		return reflect.Value{}, false
	}

	var aa []reflect.Value
	for _, arg := range args {
		aa = append(aa, reflect.ValueOf(arg))
	}

	ss := method.Call(aa)
	if len(ss) != 1 {
		return reflect.Value{}, false
	}

	return ss[0], true
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

			n, err := f.setSlice(append([]string{argname}, vals...))
			*i = *i + n
			return err
		} else {
			return f.setValue(argname)
		}
	}

	// if f.mode {
	// 	return f.setValue(val)
	// }

	if f.v.Kind() == reflect.Slice {
		if val != "" {
			_, err := f.setSlice(strings.Split(val, ","))
			return err
		}

		n, err := f.setSlice(vals)
		*i = *i + n
		return err
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

func (f *field) setSlice(vals []string) (nvals int, err error) {
	if len(f.opt) > 0 && !f.checkOptSlice(vals) {
		return 0, fmt.Errorf("impossible values %v, you should choose from %v", vals, f.opt)
	}

	nvals = len(vals)

	var n int
	var s string
	switch f.v.Interface().(type) {

	case []string:
		f.v.Set(reflect.ValueOf(vals))

	case []int:
		var ints []int
		for n, s = range vals {
			i, err := strconv.Atoi(s)
			if err != nil {
				nvals = n
				break
				// return n, err
			}
			ints = append(ints, i)
		}
		f.v.Set(reflect.ValueOf(ints))

	case []float32:
		var floats []float32
		for n, s = range vals {
			f, err := strconv.ParseFloat(s, 32)
			if err != nil {
				nvals = n
				break
				// return err
			}
			floats = append(floats, float32(f))
		}
		f.v.Set(reflect.ValueOf(floats))

	case []float64:
		var floats []float64
		for n, s = range vals {
			f, err := strconv.ParseFloat(s, 64)
			if err != nil {
				nvals = n
				break
				// return err
			}
			floats = append(floats, f)
		}
		f.v.Set(reflect.ValueOf(floats))

	case []time.Duration:
		var durs []time.Duration
		for n, s = range vals {
			d, err := time.ParseDuration(s)
			if err != nil {
				nvals = n
				break
				// return err
			}
			durs = append(durs, d)
		}
		f.v.Set(reflect.ValueOf(durs))
	}

	return
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
	var argVal []string
	if strings.Contains(s, "=") {
		argVal = strings.SplitN(s, "=", 2)

		return argVal[0], trim(argVal[1])
	}

	if len(s) > 2 && argShort.MatchString(s) {
		f, err := uf.lookupArgByShort(s[:2])
		if err == nil && f.v.Kind() != reflect.String {
			return s[:2], s[2:]
		}
	}

	return trim(s), ""
}

func trim(s string) string {
	if s[0] == '"' && s[len(s)-1] == '"' {
		s = strings.Trim(s, "\"")
	}
	if s[0] == '\'' && s[len(s)-1] == '\'' {
		s = strings.Trim(s, "'")
	}
	if s[0] == '`' && s[len(s)-1] == '`' {
		s = strings.Trim(s, "`")
	}

	return s
}

func getNextValues(osArgs []string, i int) (vals []string) {

	i++
	for ; i < len(osArgs); i++ {
		s := trim(osArgs[i])
		var err error
		switch {
		case argLong.MatchString(s):
			_, err = uf.lookupArgByLong(s)
		case argShort.MatchString(s):
			_, err = uf.lookupArgByShort(s)
		case osArgs[i] == "--":
			err = errors.New("stop")
		default:
			err = errors.New("blank")
		}

		// if err is nil than this arg is not value
		if err == nil {
			return
		}

		vals = append(vals, s)
	}

	return
}

func contains(ss []string, s string) bool {
	for _, _s := range ss {
		if _s == s {
			return true
		}
	}
	return false
}
