package gofe

import (
	"reflect"
)

type Testing interface {
	Errorf(string, ...interface{})
}

type StepFunc interface{}

type Steps map[string]StepFunc

func NewSteps() Steps {
	return make(map[string]StepFunc)
}

func (s Steps) Steps(name string, fn StepFunc) {
	s[name] = fn
}

type step struct {
	fn StepFunc
	v  []interface{}
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

func (f Feature) findStep(name string) (reflect.Value, int, error) {
	var stepFunc StepFunc

	for _, v := range f.Steps {
		if fn, ok := v[name]; ok {
			stepFunc = fn

			break // step found
		}
	}
	if stepFunc == nil {
		return reflect.Value{}, 0, nil
	}

	s := reflect.ValueOf(stepFunc).Call([]reflect.Value{
		reflect.ValueOf(f.t),
	})

	// TODO check count for len == 1

	fn := s[0]
	to := reflect.TypeOf(fn.Interface())
	return fn, to.NumIn(), nil
}

func (f Feature) Step(name string, v ...interface{}) {
	fn, n, err := f.findStep(name)
	if err != nil {
		// TODO handle
	}

	argv := make([]reflect.Value, n)
	for i := 0; i < len(argv); i++ {
		argv[i] = reflect.ValueOf(v[i])
	}

	fn.Call(argv)
}
