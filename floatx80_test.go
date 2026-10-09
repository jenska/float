package float_test

import (
	"fmt"

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
