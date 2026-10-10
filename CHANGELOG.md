# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.3.0] - 2026-10-10

### Added
- `Env`, a floating-point environment with its own rounding mode, rounding precision, tininess detection, default NaN, exception flags and handler. Every operation that rounds or raises exceptions is also an `Env` method, so separate goroutines can compute concurrently with separate `Env`s. The X80 methods and package functions are unchanged and use the package-level settings

### Changed
- `Trunc`, `ToInt16`, `ToInt8`, `Sincos` and the transcendental functions at reduced precision no longer temporarily modify the package-level rounding mode, exception flags and handler

## [1.1.0] - 2026-10-09

### Added
- Transcendental functions `Exp`, `Exp2`, `Exp10`, `Expm1`, `Log2`, `Log10`, `Log1p`, `Sin`, `Cos`, `Tan`, `Sincos`, `Asin`, `Acos`, `Sinh`, `Cosh`, `Tanh`, `Atanh`, with argument reduction exact for every X80 value
- `Mod` (truncated remainder), `RemQuo`/`ModQuo` (with quotient bits), `Scale`, `GetExp`, `GetMan`, `SglMul`, `SglDiv`, `Trunc`, `RoundToPrecision`, `Abs`, `Neg`
- `ToInt16`, `ToInt8`; `NewFromFloat32Bits`, `NewFromFloat64Bits`, `ToFloat32Bits`, `ToFloat64Bits`, which preserve NaN payloads
- `NewFromBits`, `Bits`, `Bytes96`, `NewFromBytes96` (68881/68882 and x87 12-byte formats), `Normalize`, `IsZero`, `IsSubnormal`, `Signbit`
- `Parse` (correctly rounded decimal and hexadecimal input), `Pow10`
- Constants `X80Ln10`, `X80Log10E`, `X80Log10Of2`; `Constant.Value` rounds the constants under the current rounding mode and precision
- `DefaultNaN` selects the NaN returned by invalid operations
- Benchmarks for every operation, cycling through random operands of each input class

### Performance
- Transcendental functions are evaluated in an internal 128-bit format and rounded once: 3-12x faster, allocation-free, and correctly rounded in practice (previously up to 5 ULP); they now honor the rounding mode
- Trigonometric argument reduction uses a generated 2/π table instead of `math/big`
- `Parse` and `Pow10` use a 128-bit fast path with an exact fallback near rounding boundaries (4-5x faster, no allocations)
- `Format`/`String` with `'e'`/`'g'` keep only the digits they need: about 9x faster for huge exponents
- `Bytes96` no longer allocates intermediate slices

### Changed
- Unnormal and pseudo-denormal operands are normalized instead of producing wrong results; the integer bit of infinities and NaNs is ignored
- `Float32ToFloatX80` converts directly instead of via float64, and NaN conversions keep their payload and raise invalid for signaling NaNs
- `Ln` and `Atan` round their result to `RoundingPrecision`
- **Go Version**: Requires Go 1.27+ (was 1.22+)
- Modernized code with `go fix` (builtin `min`/`max`, range-over-int, `testing.B.Loop`)
- CI reads the Go version from `go.mod`; dropped the deprecated `golint`
- Updated GitHub Actions to current majors; release workflow uses `gh release create` instead of the archived `create-release`/`upload-release-asset` actions
- `String()` uses the new `'g'` verb with 21 significant digits (enough to round-trip) instead of 30 fixed decimals
- Removed committed build artifacts (`coverage.html`, `floatx80`)

### Fixed
- `Mul` dropped the top product bit whenever it was set (e.g. `1.5*1.5` returned `0.625`)
- `Div`: `x/0` raised invalid and `0/0` raised divide-by-zero (swapped); sticky bit lost in rounding
- `Sqrt` of a negative number returned its input instead of NaN with invalid
- `Le`/`LeQuiet` returned true for `1 <= -1`; `Gt`/`Ge`/`GtQuiet`/`GeQuiet` returned true for NaN
- `Eq` reported zero equal to the smallest denormal
- `Rem` rounded ties to the wrong quotient (e.g. `7 rem 2` returned `1` instead of `-1`)
- `ToInt64RoundZero` returned `-1` for `1` and `MaxInt64` for negative overflow and for `-2^63`
- `ToFloat64` rounded ties to `0`/`1` instead of to even, and overflowed to Inf in directed rounding modes
- `ToFloat32` rounded twice via float64; it now rounds once
- Reduced-precision (32/64) overflow in directed modes returned the wrong largest finite value
- `X80Pi`, `X80E`, `X80Ln2`, `X80Log2E` and `X80Sqrt2` had only float64 precision; `X80Sqrt2` was negative
- `Ln` and `Atan` were accurate to only about two digits; they now use range reduction and are accurate to a few ulp, and no longer leak intermediate exception flags
- `Ln(-0)` returns -Inf with divide-by-zero; `Ln`/`Atan` propagate NaN payloads
- `Format` printed infinities as `NaN` and garbled exponents of 1000 or more
- `NewFromBytes` always panicked; `Bytes(binary.LittleEndian)` did not use the x87 memory layout
- Release workflow: wrong `README.md`/`go.sum` paths, missing step id, and `refs/tags/…` in asset names

## [1.0.0] - 2026-03-29

### Added
- **Core Library**: Complete 80-bit IEEE 754 extended double precision floating-point implementation
- **Arithmetic Operations**: Full set of operations (add, subtract, multiply, divide, square root, natural logarithm, arctangent)
- **Comparison Operations**: Complete comparison suite (equal, less than, greater than, etc.)
- **Type Conversions**: Conversions to/from int32, int64, float32, float64
- **Exception Handling**: IEEE 754 compliant exception handling with customizable callbacks
- **Comprehensive Testing**: Unit tests with 48.2% code coverage
- **Performance Benchmarks**: Extensive benchmark suite for performance validation
- **Documentation**: Complete API reference, usage examples, and performance notes
- **CI/CD Pipeline**: GitHub Actions workflows for testing, linting, and releases
- **Development Tools**: Makefile with common development targets
- **Security Analysis**: CodeQL integration for automated security scanning
- **Dependency Management**: Automated dependency updates via Dependabot

### Features
- **Precision**: 80-bit extended precision with 64-bit mantissa
- **Compliance**: Full IEEE 754 standard implementation
- **Performance**: Optimized bit-level operations
- **Reliability**: Comprehensive error handling and edge case testing
- **Maintainability**: Well-documented code with professional structure

### Technical Details
- **Go Version**: Requires Go 1.22+
- **Architecture**: Cross-platform (Linux, macOS, Windows)
- **Testing**: Multi-version Go testing (1.21, 1.22, 1.23)
- **Coverage**: 48.2% test coverage with HTML reports
- **Linting**: Automated code quality checks (vet, golint, staticcheck)

### Infrastructure
- **GitHub Actions**: Complete CI/CD pipeline
- **Release Automation**: Automated GitHub releases on version tags
- **Documentation**: Hosted on pkg.go.dev
- **Coverage**: Integrated with Codecov
- **Security**: Weekly CodeQL security scans

---

## [0.1] - 2026-03-XX

### Added
- Initial implementation of 80-bit floating-point arithmetic
- Basic operations and type definitions
- Initial test suite
- Basic documentation

---

[1.3.0]: https://github.com/jenska/float/releases/tag/v1.3.0
[1.1.0]: https://github.com/jenska/float/releases/tag/v1.1.0
[1.0.0]: https://github.com/jenska/float/releases/tag/v1.0.0
[0.1]: https://github.com/jenska/float/releases/tag/0.1