// Package float provides a software implementation of 80-bit IEEE 754 extended
// double precision floating-point arithmetic.
//
// This package implements the X80 type which represents 80-bit extended precision
// floating-point numbers with 1 sign bit, 15 exponent bits, and 64 mantissa bits
// (1 integer bit + 63 fraction bits).
//
// The implementation is based on the SoftFloat library and provides full IEEE 754
// compliance including proper handling of special values (NaN, infinity, denormals)
// and exception conditions.
//
// Basic usage:
//
//	import "github.com/jenska/float"
//
//	// Create values
//	a := float.X80Pi
//	b := float.NewFromFloat64(3.14159)
//
//	// Perform operations
//	sum := a.Add(b)
//	product := a.Mul(b)
//
//	// Handle exceptions
//	float.SetExceptionHandler(func(exc int) {
//	    log.Printf("FP exception: %x", exc)
//	})
//
// The X80 methods and package functions use the package-level settings
// RoundingMode, RoundingPrecision, DetectTininess and DefaultNaN and the
// package-level exception flags, and are not safe for concurrent use.  An Env
// holds these settings and flags for one goroutine:
//
//	e := &float.Env{RoundingMode: float.RoundUp}
//	third := e.Div(float.X80One, float.Int64ToFloatX80(3))
//	inexact := e.Exception&float.ExceptionInexact != 0
//
// Unnormal (nonzero exponent, integer bit clear) and pseudo-denormal (zero
// exponent, integer bit set) encodings are accepted as operands and
// normalized, as on the Motorola 68881/68882. For infinities and NaNs the
// integer bit is ignored.
//
// For more examples, see the README.md file.
package float

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/bits"
	"strconv"
)

type (
	// X80 represents the 80-bit extended double precision floating-point type
	X80 struct {
		// Sign and exponent.
		//
		//    1 bit:   sign
		//    15 bits: exponent
		high uint16
		// Integer part and fraction.
		//
		//    1 bit:   integer part
		//    63 bits: fraction
		low uint64
	}
)

// Software IEC/IEEE floating-point underflow tininess-detection mode.
const (
	TininessAfterRounding  = 0
	TininessBeforeRounding = 1
)

// DetectTininess tininess-detection mode.
var DetectTininess = TininessAfterRounding

// Software IEC/IEEE floating-point rounding mode.
const (
	RoundNearestEven = 0
	RoundToZero      = 1
	RoundDown        = 2
	RoundUp          = 3
)

// RoundingMode Software IEC/IEEE floating-point rounding mode.
var RoundingMode = RoundNearestEven

// Software IEC/IEEE floating-point exception flags.
const (
	ExceptionInvalid   = 0x01
	ExceptionDenormal  = 0x02
	ExceptionDivbyzero = 0x04
	ExceptionOverflow  = 0x08
	ExceptionUnderflow = 0x10
	ExceptionInexact   = 0x20
)

// Exception Software IEC/IEEE floating-point exception flags.
var Exception int

// RoundingPrecision Software IEC/IEEE extended double-precision rounding precision.  Valid
// values are 32, 64, and 80.
var RoundingPrecision = 80

// "constants" for X80 format, correctly rounded to 64 significand bits
var (
	X80Zero     = newFromHexString("00000000000000000000") // 0
	X80One      = newFromHexString("3FFF8000000000000000") // 1
	X80MinusOne = newFromHexString("BFFF8000000000000000") // -1
	X80E        = newFromHexString("4000ADF85458A2BB4A9B") // e
	X80Pi       = newFromHexString("4000C90FDAA22168C235") // pi
	X80Sqrt2    = newFromHexString("3FFFB504F333F9DE6484") // sqrt(2)
	X80Log2E    = newFromHexString("3FFFB8AA3B295C17F0BC") // Log2(e)
	X80Ln2      = newFromHexString("3FFEB17217F7D1CF79AC") // Ln(2)
	X80InfPos   = newFromHexString("7FFF8000000000000000") // inf+
	X80InfNeg   = newFromHexString("FFFF8000000000000000") // inf-
	X80NaN      = newFromHexString("7FFFC000000000000000") // NaN
)

// DefaultNaN is the quiet NaN returned by invalid operations such as 0/0,
// Inf-Inf or Sqrt(-1) when no operand is a NaN. The x87 uses FFFFC000000000000000;
// the 68881/68882 uses 7FFFFFFFFFFFFFFFFFFF.
var DefaultNaN = X80NaN

// ExceptionHandler is a function that gets called when a floating-point exception occurs.
type ExceptionHandler func(exception int)

// Global exception handler. Can be set by users to customize error handling.
var exceptionHandler ExceptionHandler

// SetExceptionHandler sets a custom handler for floating-point exceptions.
// The handler function will be called whenever an exception is raised.
// If no handler is set, exceptions are silently accumulated in the Exception variable.
func SetExceptionHandler(handler ExceptionHandler) {
	exceptionHandler = handler
}

// GetExceptionHandler returns the current exception handler.
func GetExceptionHandler() ExceptionHandler {
	return exceptionHandler
}

// ClearExceptions clears all pending exceptions.
func ClearExceptions() {
	Exception = 0
}

// GetExceptions returns the current exception flags.
func GetExceptions() int {
	return Exception
}

// HasException checks if a specific exception flag is set.
func HasException(flag int) bool {
	return (Exception & flag) != 0
}

// HasAnyException checks if any exception flags are set.
func HasAnyException() bool {
	return Exception != 0
}

// ClearException clears a specific exception flag.
func ClearException(flag int) {
	Exception &^= flag
}

// Raise any or all of the software IEC/IEEE floating-point exception flags.
func Raise(x int) {
	Exception |= x
	if exceptionHandler != nil {
		exceptionHandler(x)
	}
}

// NewFromFloat64 returns the result of converting the double-precision floating-point value
// `a' to the extended double-precision floating-point format.  The conversion
// is performed according to the IEC/IEEE Standard for Binary Floating-Point
// Arithmetic.
func (e *Env) NewFromFloat64(a float64) X80 {
	return e.Float64ToFloatX80(a)
}

// Bytes returns the 10-byte memory representation of an extended double
// precision float. With binary.LittleEndian the layout matches the x87
// (significand first, then sign and exponent); with binary.BigEndian it is the
// byte-reversed form (sign and exponent first, then significand).
func (a X80) Bytes(order binary.ByteOrder) []byte {
	b := make([]byte, 10)
	if isLittleEndian(order) {
		order.PutUint64(b[0:8], a.low)
		order.PutUint16(b[8:10], a.high)
	} else {
		order.PutUint16(b[0:2], a.high)
		order.PutUint64(b[2:10], a.low)
	}
	return b
}

// NewFromBytes returns a new extended double precision float from its 10-byte
// memory representation, as produced by Bytes. It panics if b is shorter
// than 10 bytes.
func NewFromBytes(b []byte, order binary.ByteOrder) X80 {
	if len(b) < 10 {
		panic(fmt.Errorf("float: NewFromBytes needs 10 bytes, got %d", len(b)))
	}
	if isLittleEndian(order) {
		return X80{high: order.Uint16(b[8:10]), low: order.Uint64(b[0:8])}
	}
	return X80{high: order.Uint16(b[0:2]), low: order.Uint64(b[2:10])}
}

func isLittleEndian(order binary.ByteOrder) bool {
	switch order {
	case binary.LittleEndian:
		return true
	case binary.BigEndian:
		return false
	}
	var probe [2]byte
	order.PutUint16(probe[:], 1)
	return probe[0] == 1
}

// NewFromBits returns the X80 value with the given sign/exponent word and
// 64-bit significand (explicit integer bit in bit 63), without any conversion.
func NewFromBits(high uint16, low uint64) X80 {
	return X80{high: high, low: low}
}

// Bits returns the sign/exponent word and the 64-bit significand of a.
func (a X80) Bits() (high uint16, low uint64) {
	return a.high, a.low
}

// Bytes96 returns the 12-byte memory representation of a, which pads the
// 10-byte value with a zero word. With binary.BigEndian this is the 68881/68882
// extended format (sign and exponent, zero padding, significand); with
// binary.LittleEndian it is the x87 12-byte long double (significand, sign and
// exponent, zero padding).
func (a X80) Bytes96(order binary.ByteOrder) []byte {
	b := make([]byte, 12)
	if isLittleEndian(order) {
		order.PutUint64(b[0:8], a.low)
		order.PutUint16(b[8:10], a.high)
	} else {
		order.PutUint16(b[0:2], a.high)
		order.PutUint64(b[4:12], a.low)
	}
	return b
}

// NewFromBytes96 returns a new extended double precision float from its 12-byte
// memory representation, as produced by Bytes96. The padding word is ignored.
// It panics if b is shorter than 12 bytes.
func NewFromBytes96(b []byte, order binary.ByteOrder) X80 {
	if len(b) < 12 {
		panic(fmt.Errorf("float: NewFromBytes96 needs 12 bytes, got %d", len(b)))
	}
	if isLittleEndian(order) {
		return NewFromBytes(b[0:10], order)
	}
	return X80{high: order.Uint16(b[0:2]), low: order.Uint64(b[4:12])}
}

// canonical returns a with unnormal and pseudo-denormal encodings normalized
// and the integer bit of infinities and NaNs set, so that the arithmetic
// routines only ever see canonical encodings.
func (a X80) canonical() X80 {
	exp := a.exp()
	switch {
	case exp == 0x7FFF:
		a.low |= 1 << 63
	case exp == 0:
		if a.low>>63 != 0 { // pseudo-denormal: same value as exponent 1
			a.high++
		}
	case a.low>>63 == 0: // unnormal
		if a.low == 0 {
			return packFloatX80(a.sign(), 0, 0)
		}
		shift := bits.LeadingZeros64(a.low)
		if shift >= exp {
			return packFloatX80(a.sign(), 0, a.low<<(exp-1))
		}
		return packFloatX80(a.sign(), exp-shift, a.low<<shift)
	}
	return a
}

// Normalize returns the canonical encoding of a: unnormals and
// pseudo-denormals are normalized and the integer bit of infinities and NaNs
// is set. The value is unchanged and no exception is raised.
func (a X80) Normalize() X80 {
	return a.canonical()
}

// IsZero reports whether a is +0 or -0.
func (a X80) IsZero() bool {
	a = a.canonical()
	return a.exp() == 0 && a.low == 0
}

// IsSubnormal reports whether a is a nonzero denormalized number.
func (a X80) IsSubnormal() bool {
	a = a.canonical()
	return a.exp() == 0 && a.low != 0
}

// Signbit reports whether the sign bit of a is set, including for -0 and NaNs.
func (a X80) Signbit() bool {
	return a.sign()
}

// Abs returns a with the sign bit cleared. Like the IEEE 754 abs operation it
// only changes the sign bit: NaNs are not quieted and no exception is raised.
func (a X80) Abs() X80 {
	a.high &^= 0x8000
	return a
}

// Neg returns a with the sign bit flipped. Like the IEEE 754 negate operation
// it only changes the sign bit: NaNs are not quieted and no exception is raised.
func (a X80) Neg() X80 {
	a.high ^= 0x8000
	return a
}

// Returns the fraction bits
func (a X80) frac() uint64 {
	return a.low
}

// Returns the exponent bits
func (a X80) exp() int {
	return int(a.high & 0x7fff)
}

// Returns true if value is negative, false otherwise
func (a X80) sign() bool {
	return (a.high >> 15) != 0
}

// newFromString returns a new 80-bit floating-point value based on s, which
// contains 20 bytes in hexadecimal format.
func newFromHexString(s string) X80 {
	if len(s) != 20 {
		panic(fmt.Errorf("invalid length of float80 hexadecimal representation, expected 20, got %d", len(s)))
	}
	high, err := strconv.ParseUint(s[:4], 16, 16)
	if err != nil {
		panic(err)
	}
	low, err := strconv.ParseUint(s[4:], 16, 64)
	if err != nil {
		panic(err)
	}
	return X80{uint16(high), low}
}

// Takes two extended double-precision floating-point values `a' and `b', one
// of which is a NaN, and returns the appropriate NaN result.  If either `a' or
// `b' is a signaling NaN, the invalid exception is raised.
func (e *Env) propagateFloatX80NaN(a, b X80) X80 {
	aIsNaN := a.IsNaN()
	aIsSignalingNaN := a.IsSignalingNaN()
	bIsNaN := b.IsNaN()
	bIsSignalingNaN := b.IsSignalingNaN()
	a.low |= 0xC000000000000000
	b.low |= 0xC000000000000000
	if aIsSignalingNaN || bIsSignalingNaN {
		e.Raise(ExceptionInvalid)
	}
	if aIsNaN {
		if aIsSignalingNaN && bIsNaN {
			return b
		}
		return a
	}
	return b
}

// IsNaN returns true if the value is NaN, otherwise false
func (a X80) IsNaN() bool {
	return (a.high&0x7fff) == 0x7fff && a.low<<1 != 0
}

// IsSignalingNaN returns true of the value is a signaling NaN, otherwise false
func (a X80) IsSignalingNaN() bool {
	aLow := a.low & ^uint64(0x4000000000000000)
	return (a.high&0x7fff) == 0x7fff && aLow<<1 != 0 && a.low == aLow
}

// IsInf returns true if the value is positive or negative infinity, otherwise false
func (a X80) IsInf() bool {
	return (a.high&0x7fff) == 0x7fff && a.low<<1 == 0
}

// Takes an abstract floating-point value having sign `zSign', exponent `zExp',
// and extended significand formed by the concatenation of `zSig0' and `zSig1',
// and returns the proper extended double-precision floating-point value
// corresponding to the abstract input.  Ordinarily, the abstract value is
// rounded and packed into the extended double-precision format, with the
// inexact exception raised if the abstract input cannot be represented
// exactly.  However, if the abstract value is too large, the overflow and
// inexact exceptions are raised and an infinity or maximal finite value is
// returned.  If the abstract value is too small, the input value is rounded to
// a subnormal number, and the underflow and inexact exceptions are raised if
// the abstract input cannot be represented exactly as a subnormal extended
// double-precision floating-point number.
//
//	If `roundingPrecision' is 32 or 64, the result is rounded to the same
//
// number of bits as single or double precision, respectively.  Otherwise, the
// result is rounded to the full precision of the extended double-precision
// format.
//
//	The input significand must be normalized or smaller.  If the input
//
// significand is not normalized, `zExp' must be 0; in that case, the result
// returned is a subnormal number, and it must not require rounding.  The
// handling of underflow and overflow follows the IEC/IEEE Standard for Binary
// Floating-Point Arithmetic.
func (e *Env) roundAndPackFloatX80(roundingPrecision int, zSign bool, zExp int, zSig0, zSig1 uint64) X80 {
	roundingMode := e.RoundingMode
	roundNearestEven := roundingMode == RoundNearestEven

	switch roundingPrecision {
	case 64:
		return e.roundAndPackFloatX80Reduced(zSign, zExp, zSig0, zSig1, 0x0000000000000400, 0x00000000000007FF)
	case 32:
		return e.roundAndPackFloatX80Reduced(zSign, zExp, zSig0, zSig1, 0x0000008000000000, 0x000000FFFFFFFFFF)
	default: // 80
		increment := int64(zSig1) < 0
		if !roundNearestEven {
			if roundingMode == RoundToZero {
				increment = false
			} else {
				if zSign {
					increment = roundingMode == RoundDown && zSig1 != 0
				} else {
					increment = roundingMode == RoundUp && zSig1 != 0
				}
			}
		}
		if 0x7FFD <= uint32(zExp-1) {
			if (0x7FFE < zExp) ||
				(zExp == 0x7FFE && zSig0 == 0xFFFFFFFFFFFFFFFF && increment) {
				return e.overflowFloatX80(zSign, 0)
			}
			if zExp <= 0 {
				isTiny := e.DetectTininess == TininessBeforeRounding ||
					zExp < 0 ||
					!increment ||
					zSig0 < 0xFFFFFFFFFFFFFFFF
				zSig0, zSig1 = shift64ExtraRightJamming(zSig0, zSig1, 1-int16(zExp))
				zExp = 0
				if isTiny && zSig1 != 0 {
					e.Raise(ExceptionUnderflow)
				}
				if zSig1 != 0 {
					e.Raise(ExceptionInexact)
				}
				if roundNearestEven {
					increment = int64(zSig1) < 0
				} else {
					if zSign {
						increment = (roundingMode == RoundDown) && zSig1 != 0
					} else {
						increment = (roundingMode == RoundUp) && zSig1 != 0
					}
				}
				if increment {
					zSig0++
					if zSig1<<1 == 0 && roundNearestEven {
						zSig0 &= ^uint64(1)
					}
					if int64(zSig0) < 0 {
						zExp = 1
					}
				}
				return packFloatX80(zSign, zExp, zSig0)
			}
		}
		if zSig1 != 0 {
			e.Raise(ExceptionInexact)
		}

		if increment {
			zSig0++
			if zSig0 == 0 {
				zExp++
				zSig0 = 0x8000000000000000
			} else {
				if zSig1<<1 == 0 && roundNearestEven {
					zSig0 &= ^uint64(1)
				}
			}
		} else {
			if zSig0 == 0 {
				zExp = 0
			}
		}
		return packFloatX80(zSign, zExp, zSig0)
	}
}

// overflowFloatX80 raises the overflow and inexact exceptions and returns
// the overflowed result with sign `zSign': an infinity, or the largest finite
// value with the significand bits `^roundMask' if the rounding mode rounds
// toward zero.
func (e *Env) overflowFloatX80(zSign bool, roundMask uint64) X80 {
	e.Raise(ExceptionOverflow | ExceptionInexact)
	roundingMode := e.RoundingMode
	if roundingMode == RoundToZero ||
		(zSign && roundingMode == RoundUp) ||
		(!zSign && roundingMode == RoundDown) {
		return packFloatX80(zSign, 0x7FFE, ^roundMask)
	}
	return packFloatX80(zSign, 0x7FFF, 0x8000000000000000)
}

// roundAndPackFloatX80Reduced is roundAndPackFloatX80 for a rounding
// precision of 32 or 64 bits, whose significand bits below the precision are
// `roundMask' with half an ulp of `roundIncrement'.
func (e *Env) roundAndPackFloatX80Reduced(zSign bool, zExp int, zSig0, zSig1, roundIncrement, roundMask uint64) X80 {
	roundingMode := e.RoundingMode
	roundNearestEven := roundingMode == RoundNearestEven
	if zSig1 != 0 {
		zSig0 |= 1
	}
	if !roundNearestEven {
		if roundingMode == RoundToZero {
			roundIncrement = 0
		} else {
			roundIncrement = roundMask
			if zSign {
				if roundingMode == RoundUp {
					roundIncrement = 0
				}
			} else {
				if roundingMode == RoundDown {
					roundIncrement = 0
				}
			}
		}
	}
	roundBits := zSig0 & roundMask
	if 0x7FFD <= uint32(zExp-1) {
		if 0x7FFE < zExp || ((zExp == 0x7FFE) && (zSig0+uint64(roundIncrement) < zSig0)) {
			return e.overflowFloatX80(zSign, roundMask)
		}
		if zExp <= 0 {
			isTiny := e.DetectTininess == TininessBeforeRounding || zExp < 0 || zSig0 <= zSig0+roundIncrement
			zSig0 = shift64RightJamming(zSig0, 1-int16(zExp))
			zExp = 0
			roundBits = zSig0 & roundMask
			if isTiny && roundBits != 0 {
				e.Raise(ExceptionUnderflow)
			}
			if roundBits != 0 {
				e.Raise(ExceptionInexact)
			}
			zSig0 += roundIncrement
			if int64(zSig0) < 0 {
				zExp = 1
			}
			roundIncrement = roundMask + 1
			if roundNearestEven && (roundBits<<1 == roundIncrement) {
				roundMask |= roundIncrement
			}
			zSig0 &= ^roundMask
			return packFloatX80(zSign, zExp, zSig0)
		}
	}
	if roundBits != 0 {
		e.Raise(ExceptionInexact)
	}
	zSig0 += roundIncrement
	if zSig0 < uint64(roundIncrement) {
		zExp++
		zSig0 = 0x8000000000000000
	}
	roundIncrement = roundMask + 1
	if roundNearestEven && (roundBits<<1 == roundIncrement) {
		roundMask |= roundIncrement
	}
	zSig0 &= ^uint64(roundMask)
	if zSig0 == 0 {
		zExp = 0
	}
	return packFloatX80(zSign, zExp, zSig0)
}

// Packs the sign `zSign', exponent `zExp', and significand `zSig' into an
// extended double-precision floating-point value, returning the result.
func packFloatX80(zSign bool, zExp int, zSig uint64) X80 {
	high := uint16(zExp)
	if zSign {
		high += 1 << 15
	}
	return X80{
		low:  zSig,
		high: high,
	}
}

// Takes an abstract floating-point value having sign `zSign', exponent
// `zExp', and significand formed by the concatenation of `zSig0' and `zSig1',
// and returns the proper extended double-precision floating-point value
// corresponding to the abstract input.  This routine is just like
// `roundAndPackFloatx80' except that the input significand does not have to be
// normalized.
func (e *Env) normalizeRoundAndPackFloatX80(roundingPrecision int, zSign bool, zExp int, zSig0, zSig1 uint64) X80 {
	if zSig0 == 0 {
		zSig0 = zSig1
		zSig1 = 0
		zExp -= 64
	}
	shiftCount := bits.LeadingZeros64(zSig0)
	zSig0, zSig1 = shortShift128Left(zSig0, zSig1, int16(shiftCount))
	zExp -= shiftCount
	return e.roundAndPackFloatX80(roundingPrecision, zSign, zExp, zSig0, zSig1)
}

// Normalizes the subnormal extended double-precision floating-point value
// represented by the denormalized significand `aSig'.
func normalizeFloatX80Subnormal(aSig uint64) (zExp int, zSig uint64) {
	shiftCount := bits.LeadingZeros64(aSig)
	zSig = aSig << shiftCount
	zExp = 1 - shiftCount
	return
}

// Takes a 64-bit fixed-point value `absZ' with binary point between bits 6
// and 7, and returns the properly rounded 32-bit integer corresponding to the
// input.  If `zSign' is 1, the input is negated before being converted to an
// integer.  Bit 63 of `absZ' must be zero.  Ordinarily, the fixed-point input
// is simply rounded to an integer, with the inexact exception raised if the
// input cannot be represented exactly as an integer.  However, if the fixed-
// point input is too large, the invalid exception is raised and the largest
// positive or negative integer is returned.
func (e *Env) roundAndPackInt32(zSign bool, absZ uint64) int32 {
	roundingMode := e.RoundingMode
	roundNearestEven := roundingMode == RoundNearestEven
	roundIncrement := uint64(0x40)

	if !roundNearestEven {
		if roundingMode == RoundToZero {
			roundIncrement = 0
		} else {
			roundIncrement = 0x7F
			if zSign {
				if roundingMode == RoundUp {
					roundIncrement = 0
				}
			} else {
				if roundingMode == RoundDown {
					roundIncrement = 0
				}
			}
		}
	}
	roundBits := absZ & 0x7F
	absZ = (absZ + roundIncrement) >> 7
	if (roundBits^0x40) == 0 && roundNearestEven {
		absZ &= ^uint64(1)
	}
	z := int32(absZ)
	if zSign {
		z = -z
	}
	if (absZ>>32) != 0 || (z != 0 && (z < 0) != zSign) {
		e.Raise(ExceptionInvalid)
		if zSign {
			return math.MinInt32
		}
		return math.MaxInt32
	}
	if roundBits != 0 {
		e.Raise(ExceptionInexact)
	}
	return z
}

// Takes the 128-bit fixed-point value formed by concatenating `absZ0' and
// `absZ1', with binary point between bits 63 and 64 (between the input words),
// and returns the properly rounded 64-bit integer corresponding to the input.
// If `zSign' is 1, the input is negated before being converted to an integer.
// Ordinarily, the fixed-point input is simply rounded to an integer, with
// the inexact exception raised if the input cannot be represented exactly as
// an integer.  However, if the fixed-point input is too large, the invalid
// exception is raised and the largest positive or negative integer is
// returned.
func (e *Env) roundAndPackInt64(zSign bool, absZ0, absZ1 uint64) int64 {
	roundingMode := e.RoundingMode
	roundNearestEven := roundingMode == RoundNearestEven
	increment := int64(absZ1) < 0

	overflow := func() int64 {
		e.Raise(ExceptionInvalid)
		if zSign {
			return math.MinInt64
		}
		return math.MaxInt64
	}

	if !roundNearestEven {
		if roundingMode == RoundToZero {
			increment = false
		} else {
			if zSign {
				increment = roundingMode == RoundDown && absZ1 != 0
			} else {
				increment = roundingMode == RoundUp && absZ1 != 0
			}
		}
	}
	if increment {
		absZ0++
		if absZ0 == 0 {
			return overflow()
		}
		if absZ1<<1 == 0 && roundNearestEven {
			absZ0 &= ^uint64(1)
		}
	}
	z := int64(absZ0)
	if zSign {
		z = -z
	}
	if z != 0 && ((z < 0) != zSign) {
		return overflow()
	}
	if absZ1 != 0 {
		e.Raise(ExceptionInexact)
	}
	return z
}

// Packs the sign `zSign', exponent `zExp', and significand `zSig' into a
// double-precision floating-point value, returning the result.  After being
// shifted into the proper positions, the three fields are simply added
// together to form the result.  This means that any integer portion of `zSig'
// will be added into the exponent.  Since a properly normalized significand
// will have an integer portion equal to 1, the `zExp' input should be 1 less
// than the desired result exponent whenever `zSig' is a complete, normalized
// significand.
func packFloat64(zSign bool, zExp int16, zSig uint64) float64 {
	return math.Float64frombits(x1(zSign)<<63 + uint64(zExp)<<52 + zSig)
}

// Takes an abstract floating-point value having sign `zSign', exponent `zExp',
// and significand `zSig', and returns the proper double-precision floating-
// point value corresponding to the abstract input.  Ordinarily, the abstract
// value is simply rounded and packed into the double-precision format, with
// the inexact exception raised if the abstract input cannot be represented
// exactly.  However, if the abstract value is too large, the overflow and
// inexact exceptions are raised and an infinity or maximal finite value is
// returned.  If the abstract value is too small, the input value is rounded
// to a subnormal number, and the underflow and inexact exceptions are raised
// if the abstract input cannot be represented exactly as a subnormal double-
// precision floating-point number.
//
//	The input significand `zSig' has its binary point between bits 62
//
// and 61, which is 10 bits to the left of the usual location.  This shifted
// significand must be normalized or smaller.  If `zSig' is not normalized,
// `zExp' must be 0; in that case, the result returned is a subnormal number,
// and it must not require rounding.  In the usual case that `zSig' is
// normalized, `zExp' must be 1 less than the “true” floating-point exponent.
// The handling of underflow and overflow follows the IEC/IEEE Standard for
// Binary Floating-Point Arithmetic.
func (e *Env) roundAndPackFloat64(zSign bool, zExp int16, zSig uint64) float64 {
	roundingMode := e.RoundingMode
	roundNearestEven := roundingMode == RoundNearestEven
	roundIncrement := int64(0x200)
	if !roundNearestEven {
		if roundingMode == RoundToZero {
			roundIncrement = 0
		} else {
			roundIncrement = 0x3FF
			if zSign {
				if roundingMode == RoundUp {
					roundIncrement = 0
				}
			} else {
				if roundingMode == RoundDown {
					roundIncrement = 0
				}
			}
		}
	}
	roundBits := zSig & 0x3FF
	if 0x7FD <= uint16(zExp) {
		if 0x7FD < zExp || (zExp == 0x7FD && int64(zSig)+roundIncrement < 0) {
			e.Raise(ExceptionOverflow | ExceptionInexact)
			bits := math.Float64bits(packFloat64(zSign, 0x7FF, 0))
			if roundIncrement == 0 {
				bits-- // largest finite value
			}
			return math.Float64frombits(bits)
		}
		if zExp < 0 {
			isTiny := e.DetectTininess == TininessBeforeRounding ||
				zExp < -1 ||
				uint64(int64(zSig)+roundIncrement) < 0x8000000000000000
			zSig = shift64RightJamming(zSig, -zExp)
			zExp = 0
			roundBits = zSig & 0x3FF
			if isTiny && roundBits != 0 {
				e.Raise(ExceptionUnderflow)
			}
		}
	}
	if roundBits != 0 {
		e.Raise(ExceptionInexact)
	}
	zSig = uint64(int64(zSig)+roundIncrement) >> 10
	if (roundBits^0x200) == 0 && roundNearestEven {
		zSig &^= 1
	}
	if zSig == 0 {
		zExp = 0
	}
	return packFloat64(zSign, zExp, zSig)
}

// Packs the sign `zSign', exponent `zExp', and significand `zSig' into a
// single-precision floating-point value, returning the result.  As with
// packFloat64, any integer portion of `zSig' is added into the exponent.
func packFloat32(zSign bool, zExp int16, zSig uint32) float32 {
	return math.Float32frombits(uint32(x1(zSign))<<31 + uint32(zExp)<<23 + zSig)
}

// Takes an abstract floating-point value having sign `zSign', exponent `zExp',
// and significand `zSig', and returns the proper single-precision floating-
// point value corresponding to the abstract input.  This is the single-
// precision counterpart of roundAndPackFloat64: the input significand `zSig'
// has its binary point between bits 30 and 29, which is 7 bits to the left of
// the usual location.
func (e *Env) roundAndPackFloat32(zSign bool, zExp int16, zSig uint32) float32 {
	roundingMode := e.RoundingMode
	roundNearestEven := roundingMode == RoundNearestEven
	roundIncrement := int32(0x40)
	if !roundNearestEven {
		if roundingMode == RoundToZero {
			roundIncrement = 0
		} else {
			roundIncrement = 0x7F
			if zSign {
				if roundingMode == RoundUp {
					roundIncrement = 0
				}
			} else {
				if roundingMode == RoundDown {
					roundIncrement = 0
				}
			}
		}
	}
	roundBits := zSig & 0x7F
	if 0xFD <= uint16(zExp) {
		if 0xFD < zExp || (zExp == 0xFD && int32(zSig)+roundIncrement < 0) {
			e.Raise(ExceptionOverflow | ExceptionInexact)
			bits := math.Float32bits(packFloat32(zSign, 0xFF, 0))
			if roundIncrement == 0 {
				bits-- // largest finite value
			}
			return math.Float32frombits(bits)
		}
		if zExp < 0 {
			isTiny := e.DetectTininess == TininessBeforeRounding ||
				zExp < -1 ||
				uint32(int32(zSig)+roundIncrement) < 0x80000000
			zSig = uint32(shift64RightJamming(uint64(zSig), -zExp))
			zExp = 0
			roundBits = zSig & 0x7F
			if isTiny && roundBits != 0 {
				e.Raise(ExceptionUnderflow)
			}
		}
	}
	if roundBits != 0 {
		e.Raise(ExceptionInexact)
	}
	zSig = uint32(int32(zSig)+roundIncrement) >> 7
	if (roundBits^0x40) == 0 && roundNearestEven {
		zSig &^= 1
	}
	if zSig == 0 {
		zExp = 0
	}
	return packFloat32(zSign, zExp, zSig)
}
