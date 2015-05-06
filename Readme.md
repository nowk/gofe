# gofe

[![Build Status](https://travis-ci.org/nowk/gofe.svg?branch=master)][0]
[![GoDoc](https://godoc.org/gopkg.in/nowk/gofe.v0?status.svg)][1]

  [0]: https://travis-ci.org/nowk/gofe
  [1]: http://godoc.org/gopkg.in/nowk/gofe.v0

Cucumber like steps within Testing.


## Install

    go get gopkg.in/nowk/gofe.v0


## Usage

### Setting up steps

	s := gofe.NewSteps()
	s.Add("my name is", func(t *testing.T) func(string) {
		return func(name string) {
			t.Logf("your name is %s", name)
		}
	})


`StepFunc`'s use a good amount of *magicks* but must implement a particular pattern to be a valid `StepFunc`. The pattern looks like:

	func(testing.TB) func()

The returning `func` can have any number of arguments as required by your step.

	s.Add("a + b = c", func(t *testing.T) func(int, int, int) {
		return func(a, b, c int) {
			if a + b != c {
				t.Errorf("%d + %d did not equal %d", a, b, c)
			}
		}
	})


### Running your steps

	fe := gofe.New(t, s)
	fe.Step("my name is", "Batman!")
	fe.Step("a + b = c", 1, 2, 3)

---

__Regex__

Steps can use Regex to match, similar to the way you would define Regex based steps in Cucumber.

A regex example of the `a + b = c` example above, would look as follows:

	s.Add(`(\d) \+ (\d) = (\d)`, func(t *testing.T) func(int, int, int) {
		return func(a, b, c, int) {
			if a + b != c {
				t.Errorf("%d + %d did not equal %d", a, b, c)
			}
		}
	}) 

And you can call this step with:

	fe.Step("1 + 2 = 3")

The Regex group matches `(\d)` will automatically parse and pass the arguments to your `StepFunc`. *Arguments are called in order of the match.*

## License

MIT
