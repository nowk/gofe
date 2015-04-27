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

type StepFunc interface{}

type Steps map[string]StepFunc

func NewSteps() Steps {
	return make(map[string]StepFunc)
}

var (
	errNotFuncTesting = fmt.Errorf("steps must implement func(Testing) func(...)")
	errMustReturnFunc = fmt.Errorf("steps must return a single func")
)

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
	if t.NumIn() != 1 {
		return errNotFuncTesting
	}
	a := t.In(0)
	if a.Kind() != reflect.Interface {
		return errNotFuncTesting
	}
	if !reflect.TypeOf(tt).Implements(a) {
		return errNotFuncTesting
	}

	if t.NumOut() != 1 {
		return errMustReturnFunc
	}
	p := t.Out(0)
	if p.Kind() != reflect.Func {
		return errMustReturnFunc
	}

	// check for *Feature argument
	f := reflect.TypeOf(&Feature{})
	for i := 0; i < p.NumIn(); i++ {
		a := p.In(i)
		if i == 0 {
			if a == reflect.TypeOf(Feature{}) {
				return fmt.Errorf("Feature must be a pointer")
			}
		} else {
			if a == f {
				return fmt.Errorf("*Feature must be the first argument")
			}
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
	t Testing

	Steps   []Steps
	Context Context
}

func New(t Testing, s ...Steps) *Feature {
	return &Feature{
		t: t,

		Steps:   s,
		Context: make(map[string]interface{}),
	}
}

func (f *Feature) SetContext(c map[string]interface{}) {
	f.Context = c
}

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

// C expands the Context objects to fn as type asserted agruments of fn. To
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
			f.t.Fatalf("%s", err)

			return // testing package will exit, this is for tests
		}

		args[i] = v
	}

	v.Call(args)
}

func (f Feature) findStep(name string) (StepFunc, error) {
	for _, v := range f.Steps {
		if f, ok := v[name]; ok {
			return f, nil
		}
	}

	return nil, fmt.Errorf("`%s`: step not found", name)
}

func (f Feature) stepfn(name string) (reflect.Value, []reflect.Value, error) {
	var fn reflect.Value

	stepFunc, err := f.findStep(name)
	if err != nil {
		return fn, nil, err
	}

	// call func(Testing) func(...)
	fn = reflect.ValueOf(stepFunc).Call([]reflect.Value{
		reflect.ValueOf(f.t),
	})[0]
	args := f.makeArgs(fn.Type())

	return fn, args, nil
}

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

func (f Feature) Step(name string, a ...interface{}) {
	fn, args, err := f.stepfn(name)
	if err != nil {
		f.t.Fatal(err)

		return // actual testing package will exit, just for testing
	}

	n := cap(args) - len(args) // number of args that comes predefined from stepfn
	for i := 0; i < n; i++ {
		args = append(args, reflect.ValueOf(a[i]))
	}

	fn.Call(args)
}

func (f Feature) And(name string, a ...interface{}) {
	f.Step(name, a...)
}

// And_ is a short to Step. The appended _ underscore is there for alignment
// with Step.
//
//		fe.Step(...)
//		fe.And_(...)
//
func (f Feature) And_(name string, a ...interface{}) {
	f.Step(name, a...)
}
