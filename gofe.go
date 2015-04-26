package gofe

import (
	"fmt"
	"reflect"
	"testing"
)

type Testing interface {
	Errorf(string, ...interface{})
}

var tt Testing = &testing.T{}

type StepFunc interface{}

type Steps map[string]StepFunc

func NewSteps() Steps {
	return make(map[string]StepFunc)
}

var (
	errNotFuncTesting = fmt.Errorf("steps must implement func(Testing) func(...)")
)

func (s Steps) Steps(name string, fn StepFunc) {
	t := reflect.TypeOf(fn)
	if t.NumIn() != 1 {
		panic(errNotFuncTesting)
	}

	a := t.In(0)
	if a.Kind() != reflect.Interface {
		panic(errNotFuncTesting)
	}
	if !reflect.TypeOf(tt).Implements(a) {
		panic(errNotFuncTesting)
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

	return nil, fmt.Errorf("%s: step not foudn", name)
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
	if t.Kind() != reflect.Func {
		panic("must be a func")
	}

	a := make([]reflect.Value, t.NumIn())

	return fn, a, nil
}

func (f Feature) Step(name string, a ...interface{}) {
	fn, args, err := f.stepfn(name)
	if err != nil {
		// TODO handle
	}

	for i := 0; i < len(a); i++ {
		args[i] = reflect.ValueOf(a[i])
	}

	fn.Call(args)
}
