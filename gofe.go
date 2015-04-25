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

func (f *Feature) Step(name string, v ...interface{}) {
	for _, s := range f.Steps {
		if fn, ok := s[name]; ok {
			t := reflect.TypeOf(fn)
			stepFunc := reflect.ValueOf(fn).Call([]reflect.Value{
				reflect.ValueOf(f.t),
			})

			t = reflect.TypeOf(stepFunc[0].Interface())

			args := make([]reflect.Value, t.NumIn())
			for i := 0; i < len(args); i++ {
				args[i] = reflect.ValueOf(v[i])
			}

			stepFunc[0].Call(args)
			return
		}
	}
}
