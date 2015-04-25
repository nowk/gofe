package gofe

type Testing interface {
	Errorf(string, ...interface{})
}

type StepFunc func(Testing) func(...interface{})

type Store struct {
	steps map[string]StepFunc
}

func NewStore() *Store {
	return &Store{
		steps: make(map[string]StepFunc),
	}
}

func (s *Store) Steps(name string, fn StepFunc) {
	s.steps[name] = fn
}

type step struct {
	fn StepFunc
	v  []interface{}
}

type Feature struct {
	stores []*Store
	steps  []step
}

func New(fe ...*Store) *Feature {
	return &Feature{
		stores: fe,
	}
}

func (f *Feature) Step(name string, v ...interface{}) {
	for _, st := range f.stores {
		if fn, ok := st.steps[name]; ok {
			f.steps = append(f.steps, step{fn, v})

			return
		}
	}
}

func (f *Feature) Test(t Testing) {
	for _, v := range f.steps {
		v.fn(t)(v.v...)
	}
}
