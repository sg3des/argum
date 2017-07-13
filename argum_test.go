package argum

import (
	"log"
	"os"
	"reflect"
	"testing"
	"time"
)

var (
	err error
	// funcCalled    bool
	// funcPtrCalled bool
)

func init() {
	log.SetFlags(log.Lshortfile)
}

type testargs struct {
	S       string `argum:"req"`
	String  string `default:"defaultStringValue"`
	Strings []string

	I    int
	Int  int
	Ints []int

	F        float32 `argum:"-f"`
	Float64  float64 `argum:"--float64"`
	Floats32 []float32
	Floats64 []float64

	B    bool
	Bool bool

	D    time.Duration `default:"1s"`
	Dur  time.Duration
	Durs []time.Duration

	Pos  string  `argum:"pos,required"`
	Pos2 float32 `argum:"positional"`
}

// func (a *testargs) SomePtrFunc() {
// 	if a == nil {
// 		log.Fatal("faled call function")
// 	}
// 	funcPtrCalled = true
// }

// func (testargs) SomeFunc() {
// 	funcCalled = true
// }

// func TestPointerArgs(t *testing.T) {
// 	log.SetFlags(log.Lshortfile)

// 	var args testargs
// 	uf, err = prepareStruct(&args)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	uf.funcRequested([]string{"somefunc", "someptrfunc"})

// 	//check called
// 	if !funcCalled {
// 		t.Error("failed called function")
// 	}
// 	if !funcPtrCalled {
// 		t.Error("failed call ptr function")
// 	}
// }

func TestPrepareArgs(t *testing.T) {
	var a testargs
	uf, err = prepareStruct(&a)
	if err != nil {
		t.Fatal(err)
	}

	fcount := reflect.TypeOf(testargs{}).NumField()
	fcount += reflect.TypeOf(&testargs{}).NumMethod()
	if len(uf.fields) != fcount {
		t.Fatalf("length of args should be %d", fcount)
	}

	str, err := uf.lookupArgByLong("--string")
	if err != nil {
		t.Error("failed prepare argument")
	}
	if str.def != "defaultStringValue" {
		t.Error("failed prepare default value")
	}

	if a.String != "defaultStringValue" {
		t.Error("failed set default value")
	}
}

func TestLookupArgByLong(t *testing.T) {
	f, err := uf.lookupArgByLong("--float64")
	if err != nil {
		t.Fatal("argument not found")
	}
	f.taken = true

	_, err = uf.lookupArgByLong("--F")
	if err == nil {
		t.Fatal("argument should not be found")
	}

	_, err = uf.lookupArgByLong("--float64")
	if err == nil {
		t.Fatal("argument must already be found")
	}
}

func TestLookupArgByShort(t *testing.T) {
	if f, err := uf.lookupArgByShort("-i"); err != nil {
		t.Error("argument not found")
	} else {
		f.taken = true
	}

	if _, err := uf.lookupArgByShort("-I"); err == nil {
		t.Error("argument should not be found")
	}

	if _, err := uf.lookupArgByShort("--Int"); err == nil {
		t.Error("argument must already be found")
	}
}

func TestLookupArgByPos(t *testing.T) {
	if f, err := uf.lookupArgByPos("str"); err != nil {
		t.Error("argument not found")
	} else {
		f.taken = true
	}

	if f, err := uf.lookupArgByPos("str"); err != nil {
		t.Error("argument not found")
	} else {
		f.taken = true
	}

	if _, err := uf.lookupArgByPos("str"); err == nil {
		t.Error("argument must already be found")
	}
}

func TestGetNextValues(t *testing.T) {
	var args struct {
		S   []string
		Str []string
	}
	uf, err = prepareStruct(&args)
	if err != nil {
		t.Error(err)
	}
	vals := getNextValues([]string{"-s", "1", "2", "--str", "4"}, 0)
	if len(vals) != 2 {
		t.Errorf("count of vals should be 2, %v", vals)
	}

	vals = getNextValues([]string{"-s", "1", "2", "--str", "4"}, 3)
	if len(vals) != 1 {
		t.Errorf("count of vals should be 1, %v", vals)
	}
}

func TestStructFieldShort(t *testing.T) {
	ok, err := uf.fields[0].structFieldShort("-s")
	if err != nil {
		t.Error(err)
	}
	if !ok {
		t.Error("faild match short arg")
	}

	ok, err = uf.fields[0].structFieldShort("-smorechars")
	if err == nil {
		t.Error("should be error")
	}
	if !ok {
		t.Error("should be founded but have error")
	}

}

func TestStructFieldLong(t *testing.T) {
	ok, err := uf.fields[1].structFieldLong("--string")
	if err != nil {
		t.Error(err)
	}
	if !ok {
		t.Error("faild match long arg")
	}

	ok, err = uf.fields[1].structFieldLong("--s")
	if err == nil {
		t.Error("should be error")
	}
	if !ok {
		t.Error("should be founded but have error")
	}
}

func TestParse(t *testing.T) {
	os.Args = []string{"testing", "-s", "str", "--string", "./longstr", "--strings", "-str0", "$str1", "/str2", "-i", "10", "--float64", "0.33", "-b", "true", "--bool", "-d", "2s", "pos-value", "0.5"}
	var a testargs
	if err := Parse(&a); err != nil {
		t.Error(err)
	}
	if a.S != "str" {
		t.Error("failed set short value")
	}
	if a.String != "./longstr" {
		t.Error("failed set long value")
	}
	if a.I != 10 {
		t.Error("failed set int")
	}
	if a.B == false {
		t.Error("failed set short bool value")
	}
	if a.Bool == false {
		t.Error("failed set long bool value")
	}
	if a.Float64 != 0.33 {
		t.Error("failed set float64 value")
	}
	if a.D != time.Duration(2e9) {
		t.Error("faild set time duration")
	}
	if a.Pos != "pos-value" {
		t.Error("failed set positional string argument")
	}
	if a.Pos2 != 0.5 {
		t.Error("failed set positional float argument")
	}
}

func TestParseWithEqualSign(t *testing.T) {
	os.Args = []string{"testing", "-s=str", "--string=longstr", "--strings=str0,str1,str2", "-i=10", "--float64=0.33", "-b=true", "--bool=true", "-d=2s", "pos-value", "0.5"}
	var a testargs
	if err := Parse(&a); err != nil {
		t.Error(err)
	}
	if a.S != "str" {
		t.Error("failed set short value")
	}
	if a.String != "longstr" {
		t.Error("failed set long value")
	}
	if a.I != 10 {
		t.Error("failed set int")
	}
	if a.B == false {
		t.Error("failed set short bool value")
	}
	if a.Bool == false {
		t.Error("failed set long bool value")
	}
	if a.Float64 != 0.33 {
		t.Error("failed set float64 value")
	}
	if a.D != time.Duration(2e9) {
		t.Error("faild set time duration")
	}
	if a.Pos != "pos-value" {
		t.Error("failed set positional string argument")
	}
	if a.Pos2 != 0.5 {
		t.Error("failed set positional float argument")
	}
}

func TestParseShort(t *testing.T) {
	log.SetFlags(log.Lshortfile)
	os.Args = []string{"testing", "-s=asd", "-i10", "-f0.33", "-btrue", "pos-argument", "--durs", "10s", "12s", "-d2s"}
	var a testargs

	if err := Parse(&a); err != nil {
		t.Error(err)
	}
	if a.I != 10 {
		t.Error("failed set int")
	}
	if a.B == false {
		t.Error("failed set short bool value")
	}
	if a.F != 0.33 {
		t.Error("failed set float64 value")
	}
	if a.D != time.Duration(2e9) {
		t.Error("faild set time duration")
	}
}

func TestParseSlices(t *testing.T) {
	os.Args = []string{"testing", "-s=s", "--ints", "0", "1", "2", "--floats32", "0.3", "1", "--floats64", "0.9898", "1.1454", "--durs", "10s", "-b", "pos"}
	var a testargs
	if err := Parse(&a); err != nil {
		t.Error(err)
	}
	if len(a.Ints) != 3 {
		t.Error("faild parse int slice")
	}
	if len(a.Floats32) != 2 {
		t.Error("faild parse floats")
	}
	if len(a.Floats64) != 2 {
		t.Error("faild parse floats")
	}
	if len(a.Durs) != 1 {
		t.Error("faild parse slice of durations")
	}
}

func TestParseSlicesWithEqualSign(t *testing.T) {
	os.Args = []string{"testing", "-s=s", "--ints=0,1,2", "--floats32=0.3,1", "--floats64=0.93,1.3", "--durs=10s", "pos"}
	var a testargs
	if err := Parse(&a); err != nil {
		t.Error(err)
	}
	if len(a.Ints) != 3 {
		t.Error("faild parse int slice")
	}
	if len(a.Floats32) != 2 {
		t.Error("faild parse floats")
	}
	if len(a.Floats64) != 2 {
		t.Error("faild parse floats")
	}
	if len(a.Durs) != 1 {
		t.Error("faild parse slice of durations")
	}
}

func TestParseWithSlicePositional(t *testing.T) {
	os.Args = []string{"testing", "pos0", "pos1", "pos2", "pos3"}

	var a struct {
		Pos   string   `argum:"pos"`
		Poses []string `argum:"pos"`
	}

	err := Parse(&a)
	if err != nil {
		t.Error(err)
	}
	if a.Pos != "pos0" {
		t.Error("faild set positional argument")
	}
	if len(a.Poses) != 3 {
		t.Error("failed set slice positional arguments")
	}
	t.Logf("%+v", a)
}

func TestParseErrors(t *testing.T) {
	os.Args = []string{"testing", "--string=str", "sd"}
	var a struct {
		R string `argum:"required"`
		S string `argum:"s"`
	}

	var err error
	uf, err = prepareStruct(&a)
	if err == nil {
		t.Error("should be error")
	}

	_, err = uf.parseArgs(os.Args)
	if err == nil {
		t.Error("should be error")
	}

	err = Parse(&a)
	if err == nil {
		t.Error("should be error")
	}

}

func TestDefaults(t *testing.T) {
	var a struct {
		S   string        `default:"str"`
		B   bool          `default:"true"`
		I   int           `default:"1"`
		F32 float32       `default:"0.1"`
		F64 float64       `default:"0.11"`
		Dur time.Duration `default:"1s"`

		Ss   []string        `default:"str0,str1"`
		Bb   []bool          `default:"true,false"`
		Ii   []int           `default:"1,2"`
		Ff32 []float32       `default:"0.1,0.2"`
		Ff64 []float64       `default:"0.11,0.22"`
		Durs []time.Duration `default:"1s,2s"`
	}

	_, err := prepareStruct(&a)
	if err != nil {
		t.Error(err)
	}

	check(t, a.S, "str", "failed set default string value")
	check(t, a.B, true, "failed set default boolean value")
	check(t, a.I, 1, "failed set default integer value")
	check(t, a.F32, float32(0.1), "failed set default float32 value")
	check(t, a.F64, 0.11, "failed set default float64 value")
	check(t, a.Dur, time.Duration(1e9), "failed set default time.Duration value")

	check(t, len(a.Ss), 2, "failed set default to slice of strings")
	check(t, len(a.Bb), 2, "failed set default to slice of booleans")
	check(t, len(a.Ii), 2, "failed set default to slice of integers")
	check(t, len(a.Ff32), 2, "failed set default to slice of float32")
	check(t, len(a.Ff64), 2, "failed set default to slice of float64")
	check(t, len(a.Durs), 2, "failed set default to slice of time durations")
}

func TestShortBooleans(t *testing.T) {
	var a struct {
		A bool
		B bool
		C bool
		D bool
		E bool
	}
	var err error
	uf, err = prepareStruct(&a)
	if err != nil {
		t.Error(err)
	}

	osArgs := splitShortBooleans([]string{"-abcde"})
	if len(osArgs) != 5 {
		t.Fatal("failed split short booleans")
	}

	_, err = uf.parseArgs([]string{"-abcde"})
	if err != nil {
		t.Error(err)
	}
}

func TestChoose(t *testing.T) {
	var args struct {
		Str   string `argum:"-s,--str,debug|normal|fast" default:"normal"`
		Slice []int  `argum:"1|2|3|4|5|6"`
		Pos   string `argum:"pos,req,one|two|three|twenty one"`
	}

	uf, err = prepareStruct(&args)
	if err != nil {
		t.Error(err)
	}
	check(t, args.Str, "normal", "failed set default value to argument with opt")

	_, err = uf.parseArgs([]string{"-s=fast", "--slice=1,2,3", "\"twenty one\""})
	if err != nil {
		t.Error(err)
	}
	check(t, args.Str, "fast", "failed set value to argument with opt")
	check(t, len(args.Slice), 3, "failed set slice to argument with opt")
	check(t, args.Pos, "twenty one", "failed set value to positional argument with opt")

	uf, _ = prepareStruct(&args)
	_, err = uf.parseArgs([]string{"-s=other", "four"})
	if err == nil {
		t.Error("should be error")
	}

	uf, _ = prepareStruct(&args)
	_, err = uf.parseArgs([]string{"--str", "other", "four"})
	if err == nil {
		t.Error("should be error")
	}

	uf, _ = prepareStruct(&args)
	_, err = uf.parseArgs([]string{"--slice", "9,7,3", "four"})
	if err == nil {
		t.Error("should be error")
	}
}

func check(t *testing.T, k, v interface{}, err string) {
	if k != v {
		t.Error(err)
	}
}

type Mode struct {
	StrVal string `argum:"pos"`
}

func TestModes(t *testing.T) {
	var args struct {
		PtrMode *Mode
		Mode    Mode
	}

	uf, err = prepareStruct(&args)
	if err != nil {
		t.Error(err)
	}
	_, err = uf.parseArgs([]string{"ptrmode", "string"})
	if err != nil {
		t.Error(err)
	}
	if args.PtrMode == nil {
		t.Error("failed create *Mode object")
		return
	}
	check(t, args.PtrMode.StrVal, "string", "failed set value to internal struct")
}
