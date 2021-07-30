# 80-bit IEEE 754 extended double precision floating-point library for Go

The float package is a software implementation of floating-point arithmetics that conforms to
the 80-bit IEEE 754 extended double precision floating-point format

This package is derived from the original SoftFloat package and was implemented as a basis for an m68881/m68882 eumlation in pure Go

## Example

```go
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

```

## Error Handling

### TODOs
- improve test coverage
- add examples
- improve error handling
- print and scan routines
- log/ln operations
- atan
- benchmarks
- extend to 128-bit IEEE 754 