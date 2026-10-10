package float

import (
	"fmt"
	"sync"
	"testing"
)

// envResults runs a fixed set of operations in e and returns their results,
// each with the exception flags it raised; flags returns and clears the
// flags that e reports to.
func envResults(e *Env, flags func() int) []string {
	three := Int64ToFloatX80(3)
	var r []string
	add := func(z any) {
		r = append(r, fmt.Sprintf("%v/%#x", z, flags()))
	}
	flags()
	add(e.Div(X80One, three))
	add(e.Sqrt(Int64ToFloatX80(2)))
	add(e.Sin(X80One))
	sin, cos := e.Sincos(three)
	add([2]X80{sin, cos})
	add(e.Exp(X80Pi))
	add(e.Trunc(Int64ToFloatX80(-5).Scale(-1)))
	add(e.RoundToInt(Int64ToFloatX80(-5).Scale(-1)))
	add(e.ToInt16(Int64ToFloatX80(5).Scale(-1)))
	add(e.ToFloat32(X80Pi))
	add(e.Pow10(-5))
	add(e.Constant(ConstLn2))
	z, _ := e.Parse("0.1")
	add(z)
	add(e.Div(X80Zero, X80Zero))
	add(e.Scale(X80One, -16446))
	return r
}

func envFlags(e *Env) func() int {
	return func() int {
		x := e.Exception
		e.Exception = 0
		return x
	}
}

func pkgFlags() int {
	x := GetExceptions()
	ClearExceptions()
	return x
}

func TestEnvConcurrent(t *testing.T) {
	type setting struct{ mode, prec int }
	var settings []setting
	for _, mode := range []int{RoundNearestEven, RoundToZero, RoundDown, RoundUp} {
		for _, prec := range []int{32, 64, 80} {
			settings = append(settings, setting{mode, prec})
		}
	}

	// an Env must agree with the package-level environment with the same
	// settings
	savedMode, savedPrec := RoundingMode, RoundingPrecision
	want := make([][]string, len(settings))
	for i, s := range settings {
		RoundingMode, RoundingPrecision = s.mode, s.prec
		want[i] = envResults(pkgEnv(), pkgFlags)
	}
	RoundingMode, RoundingPrecision = savedMode, savedPrec

	var wg sync.WaitGroup
	for range 8 {
		for i, s := range settings {
			wg.Go(func() {
				for range 20 {
					e := &Env{RoundingMode: s.mode, RoundingPrecision: s.prec}
					got := envResults(e, envFlags(e))
					for j := range got {
						if got[j] != want[i][j] {
							t.Errorf("mode %d, precision %d, operation %d: got %s, want %s",
								s.mode, s.prec, j, got[j], want[i][j])
							return
						}
					}
				}
			})
		}
	}
	wg.Wait()
	if x := GetExceptions(); x != 0 {
		t.Errorf("Env operations raised package-level exceptions %#x", x)
	}
}

func TestEnvZeroValue(t *testing.T) {
	var raised []int
	e := Env{Handler: func(x int) { raised = append(raised, x) }}
	if z := e.Sqrt(X80MinusOne); z != X80NaN {
		t.Errorf("Sqrt(-1) = %v, want X80NaN", z.Internal())
	}
	if e.Exception != ExceptionInvalid || len(raised) != 1 || raised[0] != ExceptionInvalid {
		t.Errorf("Exception = %#x, handler got %v, want invalid", e.Exception, raised)
	}
	e.DefaultNaN = NewFromBits(0x7FFF, 0xFFFFFFFFFFFFFFFF)
	if z := e.Sqrt(X80MinusOne); z != e.DefaultNaN {
		t.Errorf("Sqrt(-1) = %v, want DefaultNaN", z.Internal())
	}
	if z := e.Div(X80One, Int64ToFloatX80(3)); z != X80One.Div(Int64ToFloatX80(3)) {
		t.Errorf("1/3 = %v, want round to nearest at 80 bits", z)
	}
}
