package gofe

import (
	"fmt"
	"reflect"
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

// SetupFunc is a func that would be run before steps to setup the steps,
// return a teardown fun if applicable
type SetupFunc func(*Feature) func()

type Steps map[string]StepFunc

func NewSteps() Steps {
	return make(map[string]StepFunc)
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

	// check for *Feature argument
	f := reflect.TypeOf(&Feature{})
	for i := 0; i < p.NumIn(); i++ {
		a := p.In(i)
		if i == 0 {
			if a == reflect.TypeOf(Feature{}) {
				return fmt.Errorf("Feature must be a pointer")
			}

			continue
		}

		if a == f {
			return fmt.Errorf("*Feature must be the first argument")
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

	s[name] = fn

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

// Setup executes a collection of SetupFuncs and returns a func to teardown any
// SetupFuncs that returned a teardown func. Teardown is done in the same order
// as the setup.
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
func (f Feature) stepFunc(s StepFunc) (reflect.Value, []reflect.Value, error) {
	fn := reflect.ValueOf(s).Call([]reflect.Value{
		reflect.ValueOf(f.T),
	})[0]
	args := f.makeArgs(fn.Type())

	return fn, args, nil
}

// makeArgs returns a cap set []reflect.Value to the number of args for the func
// returned by calling func(Testing).
func (f *Feature) makeArgs(t reflect.Type) []reflect.Value {
	n := t.NumIn()
	if n == 0 {
		return nil
	}

	a := make([]reflect.Value, 0, n)

	// if first arg *Feature, inject it
	if t.In(0) == reflect.TypeOf(f) {
		a = a[:1]
		a[0] = reflect.ValueOf(f)
	}

	return a
}

// call relfects a StepFunc and calls it with any applicable arguments
func (f Feature) call(s StepFunc, a ...interface{}) {
	fn, args, err := f.stepFunc(s)
	if err != nil {
		f.T.Fatal(err)

		return
	}

	n := cap(args) - len(args) // number of args that comes predefined from stepfn
	for i := 0; i < n; i++ {
		args = append(args, reflect.ValueOf(a[i]))
	}

	fn.Call(args)
}

// Stepf calls a given StepFunc directly, with any addition arguments.
func (f Feature) Stepf(s StepFunc, a ...interface{}) {
	err := checkStep(s)
	if err != nil {
		f.T.Fatal(err)

		return
	}

	f.call(s, a...)
}

func findStep(name string, steps []Steps) StepFunc {
	for _, v := range steps {
		if f, ok := v[name]; ok {
			return f
		}
	}

	return nil
}

// Step looks up a step by name and calls it given any additional arguments
func (f Feature) Step(name string, a ...interface{}) {
	s := findStep(name, f.Steps)
	if s == nil {
		f.T.Fatalf("`%s`: step not found", name)

		return // actual testing package will exit, just for testing
	}

	f.call(s, a...)
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
