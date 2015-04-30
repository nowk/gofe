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
	s.Add("a \\+ b = 4", func(t Testing) func(int, int) {
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

	type NotATestingInterface interface {
		Foo()
	}

	assert.Panic(t, str, func() {
		s.Add("a step", func(t NotATestingInterface) {
			//
		})
	})
}

func TestStepsFuncCanBeTestingItselfInsteadOfInterfaceImpl(t *testing.T) {
	ok := false

	s := NewSteps()
	s.Add("a step", func(t *testing.T) func() {
		return func() {
			ok = true

			assert.TypeOf(t, "*testing.T", t)
		}
	})

	fe := New(t, s)
	fe.Step("a step")

	assert.True(t, ok)
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
	s.Add("a \\+ b = n", func(t Testing) func() {
		return func() {}
	})

	assert.Panic(t, "step `a \\+ b = n` already exists", func() {
		s.Add("a \\+ b = n", func(t Testing) func() {
			return func() {}
		})
	})
}

func TestStepNotFound(t *testing.T) {
	tT := &tTesting{}

	fe := New(tT, NewSteps())
	fe.Step("some step")

	assert.Equal(t, "`some step`: step not found", tT.fatalfs[0])
}

func TestFeatureIsPassedAsFirstArgumentIfDefined(t *testing.T) {
	tT := &tTesting{}

	s := NewSteps()
	s.Add("Batman's first name is ...", func(t Testing) func(*Step) {
		return func(s *Step) {
			v, _ := s.Context.Get("first_name")
			t.Errorf("Batman's first name is %s", v.(string))
		}
	})
	s.Add("Batman's full name is ...", func(t Testing) func(*Step, string) {
		return func(s *Step, last string) {
			v, _ := s.Context.Get("first_name")
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

func TestStepCanOnlyBeTheFirstArg(t *testing.T) {
	s := NewSteps()
	assert.Panic(t, "*Step must be the first argument", func() {
		s.Add("a step", func(t Testing) func(string, *Step) {
			return func(a string, s *Step) {
				//
			}
		})
	})
}

func TestStepMustBeAPointer(t *testing.T) {
	s := NewSteps()
	assert.Panic(t, "Step must be a pointer", func() {
		s.Add("a step", func(t Testing) func(Step) {
			return func(s Step) {
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
	s.Add("outer", func(t Testing) func(*Step, string, string) {
		return func(s *Step, a, b string) {
			s.Step("inner", a)
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
	s.Add("a step", func(t Testing) func(*Step) {
		return func(s *Step) {
			s.C([]string{"a", "b", "u"}, func(a, b string, u *User) {
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
	s.Add("a step", func(t Testing) func(*Step) {
		return func(s *Step) {
			s.C([]string{"u"}, func(u User) {
				//
			})
		}
	})
	s.Add("another step", func(t Testing) func(*Step) {
		return func(s *Step) {
			s.C(nil, func(u User) {
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
	s.Add("a step", func(t Testing) func(*Step) {
		return func(s *Step) {
			s.C([]string{"User"}, func(u *User) {
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
	s.Add("a step", func(t Testing) func(*Step) {
		return func(s *Step) {
			s.C(nil, func(a string, u *User, n int) {
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

func TestSetupAllowsSetupAndTeardown(t *testing.T) {
	tT := &tTesting{}

	s := NewSteps()
	s.Add("a step", func(t Testing) func(*Step) {
		return func(s *Step) {
			s.C(nil, func(a string, n int) {
				t.Logf("a: %s, n: %d", a, n)
			})
		}
	})

	fe := New(tT, s)
	td := fe.Setup(func(f *Feature) func() {
		f.Context["a"] = "a"
		f.Context["n"] = 9

		f.T.Logf("You got mail")

		return nil
	}, func(f *Feature) func() {
		return func() {
			f.T.Logf("Goodbye!")
		}
	})

	fe.Step("a step")

	td()

	assert.Equal(t, "You got mail", tT.logfs[0])
	assert.Equal(t, "a: a, n: 9", tT.logfs[1])
	assert.Equal(t, "Goodbye!", tT.logfs[2])
}

func TestStepfExecutesAStepFuncDirectly(t *testing.T) {
	tT := &tTesting{}

	aStep := func(t Testing) func(*Step, int) {
		return func(s *Step, n int) {
			s.C(nil, func(a string) {
				t.Logf("a: %s, n: %d", a, n)
			})
		}
	}

	fe := New(tT)
	fe.SetContext(map[string]interface{}{
		"a": "a",
	})
	fe.Stepf(aStep, 9)

	assert.Equal(t, "a: a, n: 9", tT.logfs[0])
}

func TestStepProvidesAccessToTheStepsName(t *testing.T) {
	tT := &tTesting{}

	aStep := func(t Testing) func(*Step) {
		return func(s *Step) {
			t.Logf("step name: `%s`", s.Name())
		}
	}

	s := NewSteps()
	s.Add("a step", func(t Testing) func(*Step) {
		return func(s *Step) {
			t.Logf("step name: `%s`", s.Name())
		}
	})

	fe := New(tT, s)
	fe.Stepf(aStep)
	fe.Step("a step")

	assert.Equal(t, "step name: ``", tT.logfs[0])
	assert.Equal(t, "step name: `a step`", tT.logfs[1])
}

func TestUseRegexToParseArgValues(t *testing.T) {
	tT := &tTesting{}

	s := NewSteps()
	s.Add("^I login as (\\w+) (\\d) times", func(t Testing) func(string, int) {
		return func(name string, n int) {
			t.Logf("string: %s, int: %d", name, n)
		}
	})
	s.Add("^It took less than (\\d+\\.\\d+) seconds",
		func(t Testing) func(float64) {
			return func(sec float64) {
				t.Logf("float: %.2f", sec)
			}
		})

	fe := New(tT, s)
	fe.Step("I login as Batman 2 times")
	fe.Step("It took less than 0.5 seconds")

	for i, v := range []string{
		"string: Batman, int: 2",
		"float: 0.50",
	} {
		assert.Equal(t, v, tT.logfs[i], v)
	}
}
