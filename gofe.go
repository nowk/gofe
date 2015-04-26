package gofe

import (
	"fmt"
	"reflect"
	"testing"
)

type Testing interface {
	Errorf(string, ...interface{})
	Fatal(...interface{})
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

	return nil
}

func (s Steps) Add(name string, fn StepFunc) {
	_, ok := s[name]
	if ok {
		panic(fmt.Sprintf("step `%s` already exists", name))
	}

	err := checkStep(fn)
	if err != nil {
		panic(err)
	}

	s[name] = fn
}

type Feature struct {
	t     Testing
	Steps []Steps
}

func New(t Testing, s ...Steps) *Feature {
	return &Feature{
		t:     t,
		Steps: s,
	}
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
	v := reflect.ValueOf(stepFunc).Call([]reflect.Value{
		reflect.ValueOf(f.t),
	})
	fn = v[0]
	t := fn.Type()
	a := make([]reflect.Value, t.NumIn())

	return fn, a, nil
}

func (f Feature) Step(name string, a ...interface{}) {
	fn, args, err := f.stepfn(name)
	if err != nil {
		f.t.Fatal(err)

		return // actual testing package will exit, just for testing
	}

	for i := 0; i < len(a); i++ {
		args[i] = reflect.ValueOf(a[i])
	}

	fn.Call(args)
}
