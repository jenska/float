package float_test

import (
	"fmt"

	"github.com/jenska/float"
)

func ExampleX80() {
	pi := float.X80Pi
	pi2 := pi.Add(pi)
	sqrtpi2 := pi2.Sqrt()
	epsilon := sqrtpi2.Mul(sqrtpi2).Sub(pi2)
	fmt.Println(epsilon.Internal())
	// Output: BFC28000000000000000
}
