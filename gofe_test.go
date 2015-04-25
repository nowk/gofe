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

func TestSteps(t *testing.T) {
	tT := new(tTesting)

	st := NewStore()
	st.Steps("1 + n = 4", func(t Testing) func(...interface{}) {
		return func(v ...interface{}) {
			n := v[0].(int)

			if 1+n != 4 {
				t.Errorf("1 + %d != 4", n)
			} else {
				t.Errorf("1 + %d == 4", n)
			}
		}
	})

	fe := New(st)
	fe.Step("1 + n = 4", 2)
	fe.Step("1 + n = 4", 3)
	fe.Step("1 + n = 4", 4)
	fe.Test(tT)

	assert.Equal(t, "1 + 2 != 4", tT.errorfs[0])
	assert.Equal(t, "1 + 3 == 4", tT.errorfs[1])
	assert.Equal(t, "1 + 4 != 4", tT.errorfs[2])
}
