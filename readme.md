# 80-bit IEEE 754 extended double precision floating-point library for Go

The float package is a software implementation of floating-point arithmetics that conforms to
the 80-bit IEEE 754 extended double precision floating-point format

This package is derived from the original SoftFloat package and was implemented as a basis for a Motorola M68881/M68882 FPU emulation in pure Go

## Installation

```bash
go get github.com/jenska/float@v1.1.0
```

### Requirements
- Go 1.27 or later

## Features

- **Full IEEE 754 Compliance**: Correctly rounded basic operations in all four rounding modes and at 32/64/80-bit rounding precision
- **Complete Arithmetic Operations**: Add, Sub, Mul, Div, Rem, Mod, Sqrt, Scale and more
- **Transcendental Functions**: Exponentials, logarithms, trigonometric, inverse trigonometric and hyperbolic functions, correctly rounded in practice
- **Type Conversions**: To/from int8, int16, int32, int64, float32, float64 (also as raw bit patterns) and decimal strings
- **Motorola 68881/68882 Support**: Every arithmetic operation of the FPU's instruction set, unnormal operands, the 96-bit memory format, constant ROM values and a configurable default NaN, as a basis for FPU emulators
- **String Formatting**: `'b'`, `'e'`, `'E'`, `'f'`, `'g'`, `'G'` verbs and a raw hexadecimal dump (`Internal`)
- **Exception Handling**: IEEE 754 exception flags with customizable handlers
- **High Performance**: Optimized bit-level operations
- **Not goroutine-safe**: rounding mode, rounding precision and exception flags are package-level state; don't use the package from several goroutines at once

## API Reference

See the [package documentation](https://pkg.go.dev/github.com/jenska/float) for details.
The 68881/68882 column names the FPU operation each function implements.

### Environment

| Name | Purpose |
|---|---|
| `RoundingMode` | `RoundNearestEven`, `RoundToZero`, `RoundDown`, `RoundUp` |
| `RoundingPrecision` | 80 (64-bit significand), 64 (53 bits) or 32 (24 bits) |
| `DetectTininess` | `TininessAfterRounding` or `TininessBeforeRounding` |
| `DefaultNaN` | NaN returned by invalid operations; set to `NewFromBits(0x7FFF, 0xFFFFFFFFFFFFFFFF)` for the 68881/68882 |
| `Exception`, `GetExceptions`, `HasException`, `ClearExceptions`, ... | Accumulated exception flags `ExceptionInvalid`, `ExceptionDivbyzero`, `ExceptionOverflow`, `ExceptionUnderflow`, `ExceptionInexact` |
| `SetExceptionHandler` | Callback for every raised exception |

### Arithmetic

| Method | Description | 68881/68882 |
|---|---|---|
| `Add`, `Sub`, `Mul`, `Div` | Basic operations | FADD, FSUB, FMUL, FDIV |
| `SglMul`, `SglDiv` | Single-precision multiply and divide with extended exponent range | FSGLMUL, FSGLDIV |
| `Rem`, `RemQuo` | IEEE remainder, optionally with the low quotient bits | FREM |
| `Mod`, `ModQuo` | Truncated remainder (C `fmod`), optionally with the low quotient bits | FMOD |
| `Sqrt` | Square root | FSQRT |
| `RoundToInt`, `Trunc` | Round to integer with the current mode / toward zero | FINT, FINTRZ |
| `RoundToPrecision` | Round to 32/64/80-bit precision | FMOVE to a register |
| `Scale`, `GetExp`, `GetMan` | Multiply by 2^n, extract exponent / significand | FSCALE, FGETEXP, FGETMAN |
| `Abs`, `Neg` | Sign-bit operations | FABS, FNEG |

### Transcendental Functions

| Method | 68881/68882 | Method | 68881/68882 |
|---|---|---|---|
| `Exp`, `Exp2`, `Exp10`, `Expm1` | FETOX, FTWOTOX, FTENTOX, FETOXM1 | `Sin`, `Cos`, `Tan`, `Sincos` | FSIN, FCOS, FTAN, FSINCOS |
| `Ln`, `Log2`, `Log10`, `Log1p` | FLOGN, FLOG2, FLOG10, FLOGNP1 | `Asin`, `Acos`, `Atan` | FASIN, FACOS, FATAN |
| `Sinh`, `Cosh`, `Tanh`, `Atanh` | FSINH, FCOSH, FTANH, FATANH | | |

### Comparison and Classification

- `Eq`, `Lt`, `Le`, `Gt`, `Ge` (invalid exception on NaN) and the `...Quiet` variants (invalid only on signaling NaN), `EqSignaling`
- `IsNaN`, `IsSignalingNaN`, `IsInf`, `IsZero`, `IsSubnormal`, `Signbit`
- `Normalize` returns the canonical encoding of unnormals and pseudo-denormals

### Conversions

| From X80 | To X80 | 68881/68882 format |
|---|---|---|
| `ToInt8`, `ToInt16`, `ToInt32`, `ToInt64` (+ `RoundZero` variants for 32/64) | `Int32ToFloatX80`, `Int64ToFloatX80` | .B, .W, .L |
| `ToFloat32Bits`, `ToFloat32` | `NewFromFloat32Bits`, `Float32ToFloatX80` | .S |
| `ToFloat64Bits`, `ToFloat64` | `NewFromFloat64Bits`, `Float64ToFloatX80`, `NewFromFloat64` | .D |
| `Bytes96` | `NewFromBytes96` | .X (12 bytes) |
| `Bytes` | `NewFromBytes` | 10-byte x87 format |
| `Bits` | `NewFromBits` | raw sign/exponent and significand |
| `Format`, `Append`, `String` | `Parse` | decimal strings, for .P packed decimal |

The `...Bits` conversions preserve NaN payloads and signaling NaNs, which Go's
`float32`/`float64` handling may not.

### Constants

- `X80Zero`, `X80One`, `X80MinusOne`, `X80InfPos`, `X80InfNeg`, `X80NaN`
- `X80Pi`, `X80E`, `X80Ln2`, `X80Ln10`, `X80Log2E`, `X80Log10E`, `X80Log10Of2`, `X80Sqrt2`, correctly rounded to nearest
- `Constant.Value()` rounds `ConstPi`, `ConstE`, `ConstLn2`, `ConstLn10`, `ConstLog2E`, `ConstLog10E` or `ConstLog10Of2` under the current rounding mode and precision
- `Pow10(n)` returns 10^n correctly rounded

Together with `X80Zero` these cover the 68881/68882 `FMOVECR` constant ROM.
Note that the ROM stores log10(2) as `3FFD9A209A84FBCFF798`, one unit below
the correctly rounded value that `X80Log10Of2` holds.

## Performance & Accuracy

### Accuracy
Add, Sub, Mul, Div, Rem, Mod, Sqrt, Scale, the conversions, `Pow10`, `Parse`
and `Constant.Value` are correctly rounded; this is checked against `math/big`
on hundreds of thousands of random operands.

The transcendental functions are evaluated with a 128-bit significand and
rounded once, in the current rounding mode and precision. Their evaluation
error is below 2^-110, so results are correctly rounded except in
astronomically rare cases: measured against 400-bit references on 54,000
random arguments over the whole domains, no result was off by more than
0.5 ULP. The trigonometric argument reduction is exact enough for every X80
value, including arguments like 10^4000.

### Performance Characteristics
- Arithmetic operations are optimized for speed while maintaining accuracy
- Series expansions are tuned for convergence speed vs precision trade-offs
- Memory layout is optimized for 64-bit architectures
- Arithmetic, conversions and transcendental functions do not allocate
- Trigonometric argument reduction multiplies by a precomputed table of 2/π (generated with `go generate`)
- `Parse` and `Pow10` fall back to exact `math/big` arithmetic only for inputs within 2^-44 ULP of a rounding boundary, or with more than 38 significant digits

### Benchmarks
Run benchmarks with:
```bash
go test -run '^$' -bench . ./...
```

Typical performance on modern hardware:
- Basic arithmetic: ~5-35 ns per operation
- Transcendental functions: ~130-400 ns per operation
- Conversions: ~3-20 ns per operation
- `Parse`, `String`: ~200-300 ns

## Examples

### Basic Usage
```go
package main

import (
	"fmt"

	"github.com/jenska/float"
)

func main() {
	pi := float.X80Pi
	fmt.Println(pi)                 // 21 significant digits
	fmt.Println(pi.Format('e', 10)) // fixed number of decimals

	x, err := float.Parse("0.1")
	if err != nil {
		panic(err)
	}
	fmt.Println(x.Mul(float.Int64ToFloatX80(3)))
	fmt.Println(float.X80One.Exp(), pi.Div(float.Int64ToFloatX80(6)).Sin())
}
```

Output:
```
3.14159265358979323851
3.1415926536e+00
0.300000000000000000011
2.71828182845904523543 0.5
```

### Exception Handling
```go
package main

import (
	"fmt"

	"github.com/jenska/float"
)

func main() {
	float.SetExceptionHandler(func(exc int) {
		if exc&float.ExceptionOverflow != 0 {
			fmt.Println("overflow")
		}
		if exc&float.ExceptionDivbyzero != 0 {
			fmt.Println("division by zero")
		}
	})
	defer float.SetExceptionHandler(nil)

	fmt.Println(float.Int64ToFloatX80(20000).Exp()) // e^20000 exceeds ~1.19e4932
	fmt.Println(float.X80Zero.Ln())

	// Flags accumulate until cleared, also without a handler.
	float.ClearExceptions()
	third := float.X80One.Div(float.Int64ToFloatX80(3))
	fmt.Println(third, float.HasException(float.ExceptionInexact))
}
```

Output:
```
overflow
+Inf
division by zero
-Inf
0.333333333333333333342 true
```

### Raw Bits and the 68881/68882 Memory Format
```go
package main

import (
	"encoding/binary"
	"fmt"

	"github.com/jenska/float"
)

func main() {
	x := float.X80Pi

	// Sign/exponent word and 64-bit significand
	high, low := x.Bits()
	fmt.Printf("%04X %016X\n", high, low)
	fmt.Println(float.NewFromBits(high, low) == x)

	// 12-byte 68881/68882 memory format (.X): exponent, padding, significand
	b := x.Bytes96(binary.BigEndian)
	fmt.Printf("% X\n", b)
	fmt.Println(float.NewFromBytes96(b, binary.BigEndian) == x)
}
```

Output:
```
4000 C90FDAA22168C235
true
40 00 00 00 C9 0F DA A2 21 68 C2 35
true
```

### Precision Compared to float64
```go
package main

import (
	"fmt"

	"github.com/jenska/float"
)

func main() {
	x64 := 1.0 / 3
	x80 := float.X80One.Div(float.Int64ToFloatX80(3))
	fmt.Printf("float64: %.21g\n", x64)
	fmt.Println("X80:    ", x80)
}
```

Output:
```
float64: 0.33333333333333331483
X80:     0.333333333333333333342
```

## Testing & Validation

### Running Tests
```bash
# Run all tests
go test ./...

# Skip the slower accuracy sweeps
go test -short ./...

# Run one group of tests, e.g. the transcendental accuracy tests
go test -run TestTranscendentalAccuracy -v

# Coverage report (coverage.out and coverage.html)
make coverage

# Run benchmarks
go test -run '^$' -bench . ./...
```

### Test Coverage
Test coverage is about 84% of statements; `make coverage` produces the current
figure.

Test categories:
- **Differential tests** (`reference_test.go`, `functions_test.go`): arithmetic, comparisons, conversions, `Parse`, `Pow10` and formatting compared with exact `math/big` results on hundreds of thousands of random operands, in all rounding modes and precisions
- **Accuracy tests** (`transcendental_test.go`): every transcendental function compared with 400-bit references over its whole domain
- **Edge cases**: NaN propagation, infinities, signed zeros, subnormals, unnormals, overflow and underflow, and the exception flags each operation raises
- **Unit tests**: per-operation tables in `operations_test.go`, `conversions_test.go` and `comparisons_test.go`

### Validation Against Reference
The implementation is validated against:
- Exact `math/big` arithmetic for every correctly rounded operation
- Independently computed 400-bit references for the transcendental functions
- Known mathematical constants (π, e, ln 2, √2, ...), recomputed from series in the tests
- IEEE 754 special cases and exception semantics

## Contributing

### Development Setup
1. Fork the repository
2. Clone your fork: `git clone https://github.com/<your-username>/float.git`
3. Install dependencies: `go mod download`
4. Run tests: `go test ./...`
5. Make your changes
6. Add tests for new functionality
7. Ensure all tests pass: `go test -cover`
8. Submit a pull request

### Code Style
- Follow standard Go formatting: `go fmt`
- Use `gofmt -s` for additional simplifications
- Add godoc comments for all exported functions/types
- Write comprehensive tests for new features
- Update documentation for API changes

### Areas for Contribution
- Performance optimizations
- More comprehensive test coverage
- Documentation improvements
- Port to other languages

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for a complete list of changes and version history.

## Related Projects

- [SoftFloat](https://github.com/ucb-bar/berkeley-softfloat-3) - Reference soft float implementation
- [Go math package](https://golang.org/pkg/math/) - Standard Go math library
- [decimal](https://github.com/shopspring/decimal) - Arbitrary precision decimal numbers

## Error Handling

The library implements IEEE 754 exception handling with the following exception flags:

- `ExceptionInvalid`: Invalid operation (e.g., sqrt of negative number, 0/0)
- `ExceptionDenormal`: Defined for x87 compatibility; never raised
- `ExceptionDivbyzero`: Division by zero
- `ExceptionOverflow`: Result too large to represent
- `ExceptionUnderflow`: Result too small to represent
- `ExceptionInexact`: Result not exactly representable

### Exception Handling API

```go
// Set a custom exception handler
float.SetExceptionHandler(func(exc int) {
    fmt.Printf("Floating-point exception: %x\n", exc)
})

// Check for exceptions
if float.HasException(float.ExceptionInvalid) {
    fmt.Println("Invalid operation occurred")
}

// Clear exceptions
float.ClearExceptions()
```

Exceptions are raised during operations but don't prevent execution. Operations return appropriate IEEE 754 values (NaN, Inf) for exceptional conditions.

## TODOs

- add more examples
