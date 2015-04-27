package gofe

import (
	"fmt"
	"testing"

	"gopkg.in/nowk/assert.v2"
)

type tTesting struct {
	Testing

	errorfs []string
	fatals  []string
	fatalfs []string
	logfs   []string
}

func (t *tTesting) Errorf(f string, v ...interface{}) {
	t.errorfs = append(t.errorfs, fmt.Sprintf(f, v...))
}

func (t *tTesting) Fatal(v ...interface{}) {
	t.fatals = append(t.fatals, fmt.Sprint(v...))
}

func (t *tTesting) Fatalf(f string, v ...interface{}) {
	t.fatalfs = append(t.fatalfs, fmt.Sprintf(f, v...))
}

func (t *tTesting) Logf(f string, v ...interface{}) {
	t.logfs = append(t.logfs, fmt.Sprintf(f, v...))
}

func TestStepsBasicTypes(t *testing.T) {
	tT := new(tTesting)

	s := NewSteps()
	s.Add("a + b = 4", func(t Testing) func(int, int) {
		return func(a, b int) {
			if a+b != 4 {
				t.Errorf("%d + %d != 4", a, b)
			} else {
				t.Errorf("%d + %d == 4", a, b)
			}
		}
	})

	fe := New(tT, s)
	fe.Step("a + b = 4", 1, 2)
	fe.Step("a + b = 4", 1, 3)
	fe.Step("a + b = 4", 1, 4)

	assert.Equal(t, "1 + 2 != 4", tT.errorfs[0])
	assert.Equal(t, "1 + 3 == 4", tT.errorfs[1])
	assert.Equal(t, "1 + 4 != 4", tT.errorfs[2])
}

func TestStepsStructTypes(t *testing.T) {
	tT := new(tTesting)

	type User struct {
		Name string
	}

	s := NewSteps()
	s.Add("user has name", func(t Testing) func(*User, string) {
		return func(u *User, name string) {
			if u.Name != name {
				t.Errorf("%s != %s", u.Name, name)
			} else {
				t.Errorf("%s == %s", u.Name, name)
			}
		}
	})

	u := &User{
		Name: "Batman",
	}

	fe := New(tT, s)
	fe.Step("user has name", u, "Batman")
	fe.Step("user has name", u, "Spongebob")

	assert.Equal(t, "Batman == Batman", tT.errorfs[0])
	assert.Equal(t, "Batman != Spongebob", tT.errorfs[1])
}

func TestStepsIsFuncWithTestingArg(t *testing.T) {
	str := "steps must implement func(Testing) func(...)"

	s := NewSteps()
	assert.Panic(t, str, func() {
		s.Add("a step", func() {
			//
		})
	})

	assert.Panic(t, str, func() {
		s.Add("a step", func(s string) {
			//
		})
	})

	assert.Panic(t, str, func() {
		s.Add("a step", func(t *testing.T) {
			//
		})
	})

	type NotATestingInterface interface {
		Foo()
	}

	assert.Panic(t, str, func() {
		s.Add("a step", func(t NotATestingInterface) {
			//
		})
	})
}

func TestStepsIsFuncThatReturnsFunc(t *testing.T) {
	str := "steps must return a single func"

	s := NewSteps()
	assert.Panic(t, str, func() {
		s.Add("a step", func(t Testing) {
			//
		})
	})

	assert.Panic(t, str, func() {
		s.Add("a step", func(t Testing) string {
			return ""
		})
	})

	assert.Panic(t, str, func() {
		s.Add("a step", func(t Testing) (func(), func()) {
			return func() {}, func() {}
		})
	})
}

func TestStepsHaveUniqueNames(t *testing.T) {
	s := NewSteps()
	s.Add("a + b = n", func(t Testing) func() {
		return func() {}
	})

	assert.Panic(t, "step `a + b = n` already exists", func() {
		s.Add("a + b = n", func(t Testing) func() {
			return func() {}
		})
	})
}

func TestStepNotFound(t *testing.T) {
	tT := &tTesting{}

	fe := New(tT, NewSteps())
	fe.Step("some step")

	assert.Equal(t, "`some step`: step not found", tT.fatals[0])
}

func TestFeatureIsPassedAsFirstArgumentIfDefined(t *testing.T) {
	tT := &tTesting{}

	s := NewSteps()
	s.Add("Batman's first name is ...", func(t Testing) func(*Feature) {
		return func(f *Feature) {
			v, _ := f.Context.Get("first_name")
			t.Errorf("Batman's first name is %s", v.(string))
		}
	})
	s.Add("Batman's full name is ...", func(t Testing) func(*Feature, string) {
		return func(f *Feature, last string) {
			v, _ := f.Context.Get("first_name")
			t.Errorf("Batman's full name is %s %s", v.(string), last)
		}
	})

	fe := New(tT, s)
	fe.SetContext(map[string]interface{}{
		"first_name": "Bruce",
	})
	fe.Step("Batman's first name is ...")
	fe.Step("Batman's full name is ...", "Wayne")

	assert.Equal(t, "Batman's first name is Bruce", tT.errorfs[0])
	assert.Equal(t, "Batman's full name is Bruce Wayne", tT.errorfs[1])
}

func TestFeatureCanOnlyBeTheFirstArg(t *testing.T) {
	s := NewSteps()
	assert.Panic(t, "*Feature must be the first argument", func() {
		s.Add("a step", func(t Testing) func(string, *Feature) {
			return func(a string, f *Feature) {
				//
			}
		})
	})
}

func TestFeatureMustBeAPointer(t *testing.T) {
	s := NewSteps()
	assert.Panic(t, "Feature must be a pointer", func() {
		s.Add("a step", func(t Testing) func(Feature) {
			return func(f Feature) {
				//
			}
		})
	})
}

func TestCallAStepWithinAStep(t *testing.T) {
	tT := &tTesting{}

	s := NewSteps()
	s.Add("inner", func(t Testing) func(string) {
		return func(a string) {
			t.Logf("inner %s", a)
		}
	})
	s.Add("outer", func(t Testing) func(*Feature, string, string) {
		return func(f *Feature, a, b string) {
			f.Step("inner", a)
			t.Logf("outer %s, %s", a, b)
		}
	})

	fe := New(tT, s)
	fe.Step("outer", "one", "two")

	assert.Equal(t, "inner one", tT.logfs[0])
	assert.Equal(t, "outer one, two", tT.logfs[1])
}

func TestCExpandsContextToFuncArgsUsingDiForOrder(t *testing.T) {
	tT := &tTesting{}

	type User struct {
		Name string
	}

	s := NewSteps()
	s.Add("a step", func(t Testing) func(*Feature) {
		return func(f *Feature) {
			f.C([]string{"a", "b", "u"}, func(a, b string, u *User) {
				t.Logf("a: %s, b: %s, u: %s", a, b, u.Name)
			})
		}
	})

	fe := New(tT, s)
	fe.SetContext(map[string]interface{}{
		"b": "b",
		"u": &User{"Batman"},
		"a": "a",
	})
	fe.Step("a step")

	assert.Equal(t, "a: a, b: b, u: Batman", tT.logfs[0])
}

func TestCArgTypesMustMatchMatchedContextType(t *testing.T) {
	tT := &tTesting{}

	type User struct {
		Name string
	}

	s := NewSteps()
	s.Add("a step", func(t Testing) func(*Feature) {
		return func(f *Feature) {
			f.C([]string{"u"}, func(u User) {
				//
			})
		}
	})
	s.Add("another step", func(t Testing) func(*Feature) {
		return func(f *Feature) {
			f.C(nil, func(u User) {
				//
			})
		}
	})

	fe := New(tT, s)
	fe.SetContext(map[string]interface{}{
		"u": &User{"Batman"},
	})
	fe.Step("a step")
	fe.Step("another step")

	assert.Equal(t, "User: invalid context injection type", tT.fatalfs[0])
	assert.Equal(t, "User: invalid context injection type", tT.fatalfs[1])
}

func TestCArgDiMustHaveAMatchingKey(t *testing.T) {
	tT := &tTesting{}

	type User struct {
		Name string
	}

	s := NewSteps()
	s.Add("a step", func(t Testing) func(*Feature) {
		return func(f *Feature) {
			f.C([]string{"User"}, func(u *User) {
				//
			})
		}
	})

	fe := New(tT, s)
	fe.SetContext(map[string]interface{}{
		"u": &User{"Batman"},
	})
	fe.Step("a step")

	assert.Equal(t, "User: invalid context injection key", tT.fatalfs[0])
}

func TestCExpandsOnTypeIfDiIsNil(t *testing.T) {
	tT := &tTesting{}

	type User struct {
		Name string
	}

	s := NewSteps()
	s.Add("a step", func(t Testing) func(*Feature) {
		return func(f *Feature) {
			f.C(nil, func(a string, u *User, n int) {
				t.Logf("a: %s, u: %s, n: %d", a, u.Name, n)
			})
		}
	})

	fe := New(tT, s)
	fe.SetContext(map[string]interface{}{
		"n": 9,
		"u": &User{"Batman"},
		"a": "a",
	})
	fe.Step("a step")

	assert.Equal(t, "a: a, u: Batman, n: 9", tT.logfs[0])
}
