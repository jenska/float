# 80-bit IEEE 754 extended double precision floating-point library for Go

The float package is a software implementation of floating-point arithmetics that conforms to
the 80-bit IEEE 754 extended double precision floating-point format

This package is derived from the original SoftFloat package and was implemented as a basis for a Motorola M68881/M68882 FPU emulation in pure Go

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
    fmt.Println(epsilon)
    // Output: -0.000000000000000000433680868994
}
```

## Error Handling

### TODOs

- improve test coverage
- add examples
- improve error handling
- log/ln operations
- atan
- benchmarks
