package float_test

import (
	"fmt"
	"sync"

	"github.com/jenska/float"
)

func ExampleX80() {
	pi := float.X80Pi
	fmt.Println(pi)
	fmt.Println(pi.Format('e', 10))

	// The square of a correctly rounded square root recovers 2*pi exactly here.
	pi2 := pi.Add(pi)
	sqrtpi2 := pi2.Sqrt()
	fmt.Println(sqrtpi2.Mul(sqrtpi2).Sub(pi2))
	// Output:
	// 3.14159265358979323851
	// 3.1415926536e+00
	// 0
}

func ExampleEnv() {
	// Each goroutine uses its own Env with its own rounding mode and flags.
	third := make([]float.X80, 2)
	inexact := make([]bool, 2)
	var wg sync.WaitGroup
	for i, mode := range []int{float.RoundDown, float.RoundUp} {
		wg.Go(func() {
			e := &float.Env{RoundingMode: mode}
			third[i] = e.Div(float.X80One, float.Int64ToFloatX80(3))
			inexact[i] = e.Exception&float.ExceptionInexact != 0
		})
	}
	wg.Wait()
	fmt.Println(third[0].Format('e', 22), inexact[0])
	fmt.Println(third[1].Format('e', 22), inexact[1])
	// Output:
	// 3.3333333333333333331526e-01 true
	// 3.3333333333333333334237e-01 true
}
