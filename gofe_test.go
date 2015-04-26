package gofe

import (
	"fmt"
	"gopkg.in/nowk/assert.v2"
	"testing"
)

type tTesting struct {
	errorfs []string
}

func (t *tTesting) Errorf(f string, v ...interface{}) {
	t.errorfs = append(t.errorfs, fmt.Sprintf(f, v...))
}

func TestStepsBasicTypes(t *testing.T) {
	tT := new(tTesting)

	s := NewSteps()
	s.Steps("a + b = 4", func(t Testing) func(int, int) {
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
	s.Steps("user has name", func(t Testing) func(*User, string) {
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
		s.Steps("a step", func() {
			//
		})
	})

	assert.Panic(t, str, func() {
		s.Steps("a step", func(s string) {
			//
		})
	})

	assert.Panic(t, str, func() {
		s.Steps("a step", func(t *testing.T) {
			//
		})
	})

	type NotATestingInterface interface {
		Foo()
	}

	assert.Panic(t, str, func() {
		s.Steps("a step", func(t NotATestingInterface) {
			//
		})
	})
}

func TestStepsIsFuncThatReturnsFunc(t *testing.T) {
	str := "steps must return a single func"

	s := NewSteps()
	assert.Panic(t, str, func() {
		s.Steps("a step", func(t Testing) {
			//
		})
	})

	assert.Panic(t, str, func() {
		s.Steps("a step", func(t Testing) string {
			return ""
		})
	})

	assert.Panic(t, str, func() {
		s.Steps("a step", func(t Testing) (func(), func()) {
			return func() {}, func() {}
		})
	})
}

func TestStepsHaveUniqueNames(t *testing.T) {
	s := NewSteps()
	s.Steps("a + b = n", func(t Testing) func() {
		return func() {}
	})

	assert.Panic(t, "step `a + b = n` already exists", func() {
		s.Steps("a + b = n", func(t Testing) func() {
			return func() {}
		})
	})
}
