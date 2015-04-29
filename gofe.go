package gofe

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"testing"
)

type Testing interface {
	Error(...interface{})
	Errorf(string, ...interface{})
	Fail()
	FailNow()
	Failed() bool
	Fatal(...interface{})
	Fatalf(string, ...interface{})
	Log(...interface{})
	Logf(string, ...interface{})
	Skip(...interface{})
	SkipNow()
	Skipf(string, ...interface{})
	Skipped() bool
}

var tt Testing = &testing.T{}

// StepFunc must implement a func(Testing) func(...) pattern
type StepFunc interface{}

// SetupFunc represents a func to setup the Feature. Providing access to the
// Feature itself to set contexts. The returning func is any teardown process
// for the given func.
type SetupFunc func(*Feature) func()

type step struct {
	name string
	fn   StepFunc
	reg  *regexp.Regexp
}

type Steps map[string]*step

func NewSteps() Steps {
	return make(Steps)
}

func checkFuncTesting(t reflect.Type) bool {
	if t.NumIn() != 1 {
		return false
	}

	a := t.In(0)
	b := reflect.TypeOf(tt)

	if a.Kind() == reflect.Interface {
		return b.Implements(a)
	}

	return a == b
}

func checkFuncTestingReturnsFunc(t reflect.Type) (reflect.Type, bool) {
	if t.NumOut() != 1 {
		return nil, false
	}

	p := t.Out(0)

	return p, p.Kind() == reflect.Func
}

// checkStep checks to make sure the StepFunc given for any step meets the
// required implmenetation of func(Testing) func(...)
//
//		s.Steps("a step", func(t Testing) func(string, string) {
//			return func(a, b string) {
//				if a != b {
//					t.Errorf("%s != %s", a, b)
//				}
//			}
//		})
//
func checkStep(fn StepFunc) error {
	t := reflect.TypeOf(fn)

	if ok := checkFuncTesting(t); !ok {
		return fmt.Errorf("steps must implement func(Testing) func(...)")
	}

	p, ok := checkFuncTestingReturnsFunc(t)
	if !ok {
		return fmt.Errorf("steps must return a single func")
	}

	// check for *Step argument
	s := reflect.TypeOf(&Step{})
	for i := 0; i < p.NumIn(); i++ {
		a := p.In(i)
		if i == 0 {
			if a == reflect.TypeOf(Step{}) {
				return fmt.Errorf("Step must be a pointer")
			}

			continue
		}

		if a == s {
			return fmt.Errorf("*Step must be the first argument")
		}
	}

	return nil
}

// Add adds a StepFunc by name. It always returns nil to allow steps to be added
// without using an init() or some sort of initialization block
func (s Steps) Add(name string, fn StepFunc) interface{} {
	_, ok := s[name]
	if ok {
		panic(fmt.Sprintf("step `%s` already exists", name))
	}

	err := checkStep(fn)
	if err != nil {
		panic(err)
	}

	s[name] = &step{
		name: name,
		fn:   fn,
		reg:  regexp.MustCompile(name),
	}

	return nil
}

type Context map[string]interface{}

func (c Context) Get(k string) (interface{}, bool) {
	v, ok := c[k]
	if !ok {
		return nil, false
	}

	return v, true
}

type Feature struct {
	T       Testing
	Steps   []Steps
	Context Context
}

func New(t Testing, s ...Steps) *Feature {
	return &Feature{
		T:       t,
		Steps:   s,
		Context: make(map[string]interface{}),
	}
}

func (f *Feature) SetContext(c map[string]interface{}) {
	f.Context = c
}

// getc looks up a context by type and then by key returning it's reflected
// value.
func (f Feature) getc(t reflect.Type, key string) (reflect.Value, error) {
	var v, null reflect.Value

	for k, c := range f.Context {
		vo := reflect.ValueOf(c)
		if vo.Type() == t {
			v = vo

			// if key == "" it's assumed the di value was nil and returning upon a
			// matched type is enough
			if k == key || key == "" {
				return v, nil
			}
		}
	}
	// no matched type was found
	if reflect.DeepEqual(v, null) {
		return v, fmt.Errorf("%s: invalid context injection type", t.Name())
	}

	return v, fmt.Errorf("%s: invalid context injection key", key)
}

// C expands the Context objects to fn as type asserted arguments of fn. To
// handle similar types, C employees an angular style Direct Injection array to
// help attempt to match the order of the arguments.
func (f Feature) C(di []string, fn interface{}) {
	v := reflect.ValueOf(fn)
	n := v.Type().NumIn()

	args := make([]reflect.Value, n)

	for i := 0; i < n; i++ {
		var k string
		if len(di) > 0 {
			k = di[i]
		}

		v, err := f.getc(v.Type().In(i), k)
		if err != nil {
			f.T.Fatalf("%s", err)

			return // testing package will exit, this is for tests
		}

		args[i] = v
	}

	v.Call(args)
}

// Setup calls SetupFuncs and returns a teardown func with any teardown funcs
// returned by the given SetupFuncs. Teardown order is FIFO.
func (f *Feature) Setup(fn ...SetupFunc) func() {
	var tds []func()

	for _, v := range fn {
		td := v(f)
		if td != nil {
			tds = append(tds, td)
		}
	}

	return func() {
		for _, v := range tds {
			v()
		}
	}
}

// stepFunc calls func(Testing) func(...)
func (f Feature) stepFunc(s StepFunc) (reflect.Value, []reflect.Value) {
	t := []reflect.Value{
		reflect.ValueOf(f.T),
	}
	fn := reflect.ValueOf(s).Call(t)[0]

	n := fn.Type().NumIn()
	if n == 0 {
		return fn, nil
	}
	args := make([]reflect.Value, 0, n)

	return fn, args
}

// Step embeds Feature and provides access to the step's name
type Step struct {
	*Feature

	name string
}

func (s Step) Name() string {
	return s.name
}

func checkParam(i interface{}, t reflect.Type) (reflect.Value, error) {
	v := reflect.ValueOf(i)

	// just return it's not a param
	if v.Type() != reflect.TypeOf(&param{}) {
		return v, nil
	}

	str := i.(*param).v

	var err error
	var p interface{}
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p, err = strconv.ParseInt(str, 0, 64)

	case reflect.Float32, reflect.Float64:
		p, err = strconv.ParseFloat(str, 64)

	case reflect.String:
		p = str

		// TODO other string -> to conversions

	default:
		// TODO handle
	}

	// last force covert to arg type
	return reflect.ValueOf(p).Convert(t), err
}

// call relfects a StepFunc and calls it with any available arguments
func (f *Feature) call(name string, s StepFunc, a ...interface{}) {
	fn, args := f.stepFunc(s)
	t := fn.Type()
	if args != nil {
		// if first arg *Step, inject it
		if t.In(0) == reflect.TypeOf(&Step{}) {
			args = args[:1]
			args[0] = reflect.ValueOf(&Step{
				Feature: f,

				name: name,
			})
		}

		l := len(args)
		n := cap(args) - l // number offset, if *Step is first arg
		for i := 0; i < n; i++ {
			v := t.In(i + l)

			p, err := checkParam(a[i], v)
			if err != nil {
				// TODO handle
			}

			args = append(args, p)
		}
	}

	fn.Call(args)
}

// Stepf calls a given StepFunc directly
func (f Feature) Stepf(fn StepFunc, a ...interface{}) {
	err := checkStep(fn)
	if err != nil {
		f.T.Fatal(err)

		return
	}

	f.call("", fn, a...)
}

type param struct {
	v string
}

// Step looks up a step by name and calls it
func (f Feature) Step(name string, a ...interface{}) {
	var fn StepFunc
	var args []interface{}

	for _, s := range f.Steps {
		for _, v := range s {
			m := v.reg.FindStringSubmatch(name)
			if len(m) > 0 {
				fn = v.fn

				for i, v := range m {
					if i > 0 {
						args = append(args, &param{v})
					}
				}

				break
			}
		}
	}

	if fn == nil {
		f.T.Fatalf("`%s`: step not found", name)

		return // actual testing package will exit, just for testing
	}

	f.call(name, fn, append(args, a...)...)
}

/*
Cucumber style methods

*/

func (f Feature) Given(name string, a ...interface{}) {
	f.Step(name, a...)
}

func (f Feature) When(name string, a ...interface{}) {
	f.Step(name, a...)
}

func (f Feature) Then(name string, a ...interface{}) {
	f.Step(name, a...)
}

func (f Feature) And(name string, a ...interface{}) {
	f.Step(name, a...)
}

/*
_ prefixed shortcuts for alignment

		fe.Given(...)
		fe.And__(...)
		fe.When_(...)
		fe.Then_(...)

*/

func (f Feature) When_(name string, a ...interface{}) {
	f.Step(name, a...)
}

func (f Feature) Then_(name string, a ...interface{}) {
	f.Step(name, a...)
}

func (f Feature) And_(name string, a ...interface{}) {
	f.Step(name, a...)
}
func (f Feature) And__(name string, a ...interface{}) {
	f.Step(name, a...)
}
