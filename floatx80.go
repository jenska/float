package float

import (
	"fmt"
	"math"
	"math/bits"
	"strconv"
)

type (
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

// Software IEC/IEEE floating-point underflow tininess-detection mode.
var DetectTininess = TininessAfterRounding

// Software IEC/IEEE floating-point rounding mode.
const (
	RoundNearestEven = 0
	RoundToZero      = 1
	RoundDown        = 2
	RoundUp          = 3
)

// Software IEC/IEEE floating-point rounding mode.
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

// Software IEC/IEEE floating-point exception flags.
var Exception int = 0

// Software IEC/IEEE extended double-precision rounding precision.  Valid
// values are 32, 64, and 80.
var RoundingPrecision = 80

// "constants" fpr X80 format
var (
	X80Zero     = newFromHexString("00000000000000000000") // 0
	X80One      = newFromHexString("3FFF8000000000000000") // 1
	X80MinusOne = newFromHexString("BFFF8000000000000000") // -1
	X80E        = newFromHexString("4000ADF85458A2BB4800") // e
	X80Pi       = newFromHexString("4000C90FDAA22168C000") // pi
	X80Sqrt2    = newFromHexString("BFFFB504F333F9DE6800") // sqrt(2)
	X80Log2E    = newFromHexString("3FFFB8AA3B295C17F000") // Log2(e)
	X80Ln2      = newFromHexString("3FFEB17217F7D1CF7800") // Ln(2)
	X80InfPos   = newFromHexString("7FFF8000000000000000") // inf+
	X80InfNeg   = newFromHexString("FFFF8000000000000000") // inf-
	X80NaN      = newFromHexString("7FFFC000000000000000") // NaN
)

// Routine to raise any or all of the software IEC/IEEE floating-point exception flags.
func Raise(x int) {
	Exception |= x
	// TODO: callback if Exception!=0
	// Do not use global var
}

func (a X80) String() string {
	return fmt.Sprintf("%04X%016X", a.high, a.low)
}

// Returns the faction bits
func (a X80) frac() uint64 {
	return a.low
}

// Returns the exponent bits
func (a X80) exp() int32 {
	return int32(a.high & 0x7fff)
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
func propagateFloatX80NaN(a, b X80) X80 {
	aIsNaN := a.IsNaN()
	aIsSignalingNaN := a.IsSignalingNaN()
	bIsNaN := b.IsNaN()
	bIsSignalingNaN := b.IsSignalingNaN()
	a.low |= 0xC000000000000000
	b.low |= 0xC000000000000000
	if aIsSignalingNaN || bIsSignalingNaN {
		Raise(ExceptionInvalid)
	}
	if aIsNaN {
		if aIsSignalingNaN && bIsNaN {
			return b
		}
		return a
	} else {
		return b
	}
}

// Returns true if the value is NaN, otherwise false
func (a X80) IsNaN() bool {
	return (a.high&0x7fff) == 0x7fff && a.low<<1 != 0
}

// Returns true of the value is a signaling NaN, otherwise false
func (a X80) IsSignalingNaN() bool {
	aLow := a.low & ^uint64(0x4000000000000000)
	return (a.high&0x7fff) == 0x7fff && aLow<<1 != 0 && a.low == aLow
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
//    If `roundingPrecision' is 32 or 64, the result is rounded to the same
// number of bits as single or double precision, respectively.  Otherwise, the
// result is rounded to the full precision of the extended double-precision
// format.
//    The input significand must be normalized or smaller.  If the input
// significand is not normalized, `zExp' must be 0; in that case, the result
// returned is a subnormal number, and it must not require rounding.  The
// handling of underflow and overflow follows the IEC/IEEE Standard for Binary
// Floating-Point Arithmetic.
func roundAndPackFloatX80(roundingPrecision int, zSign bool, zExp int32, zSig0, zSig1 uint64) X80 {
	roundingMode := RoundingMode
	roundNearestEven := roundingMode == RoundNearestEven

	overflow := func(roundMask uint64) X80 {
		Raise(ExceptionOverflow | ExceptionInexact)
		if roundingMode == RoundToZero ||
			(zSign && roundingMode == RoundUp) ||
			(!zSign && roundingMode == RoundDown) {
			return packFloatX80(zSign, 0x7FFE, ^roundMask)
		}
		return packFloatX80(zSign, 0x7FFF, 0x8000000000000000)
	}

	precision64 := func(roundIncrement, roundMask uint64) X80 {
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
				return overflow(uint64(roundingMode))
			}
			if zExp <= 0 {
				isTiny := DetectTininess == TininessBeforeRounding || zExp < 0 || zSig0 <= zSig0+roundIncrement
				zSig0 = shift64RightJamming(zSig0, 1-int16(zExp))
				zExp = 0
				roundBits = zSig0 & roundMask
				if isTiny && roundBits != 0 {
					Raise(ExceptionUnderflow)
				}
				if roundBits != 0 {
					Raise(ExceptionInexact)
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
			Raise(ExceptionInexact)
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

	switch roundingPrecision {
	case 64:
		return precision64(0x0000000000000400, 0x00000000000007FF)
	case 32:
		return precision64(0x0000008000000000, 0x000000FFFFFFFFFF)
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
				return overflow(0)
			}
			if zExp <= 0 {
				isTiny := DetectTininess == TininessBeforeRounding ||
					zExp < 0 ||
					!increment ||
					zSig0 < 0xFFFFFFFFFFFFFFFF
				zSig0, zSig1 = shift64ExtraRightJamming(zSig0, zSig1, 1-int16(zExp))
				zExp = 0
				if isTiny && zSig1 != 0 {
					Raise(ExceptionUnderflow)
				}
				if zSig1 != 0 {
					Raise(ExceptionInexact)
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
			Raise(ExceptionInexact)
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

// Packs the sign `zSign', exponent `zExp', and significand `zSig' into an
// extended double-precision floating-point value, returning the result.
func packFloatX80(zSign bool, zExp int32, zSig uint64) X80 {
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
//`zExp', and significand formed by the concatenation of `zSig0' and `zSig1',
// and returns the proper extended double-precision floating-point value
//corresponding to the abstract input.  This routine is just like
//`roundAndPackFloatx80' except that the input significand does not have to be
// normalized.
func normalizeRoundAndPackFloatX80(roundingPrecision int, zSign bool, zExp int32, zSig0, zSig1 uint64) X80 {
	if zSig0 == 0 {
		zSig0 = zSig1
		zSig1 = 0
		zExp -= 64
	}
	shiftCount := bits.LeadingZeros64(zSig0)
	zSig0, zSig1 = shortShift128Left(zSig0, zSig1, int16(shiftCount))
	zExp -= int32(shiftCount)
	return roundAndPackFloatX80(roundingPrecision, zSign, zExp, zSig0, zSig1)
}

// Normalizes the subnormal extended double-precision floating-point value
// represented by the denormalized significand `aSig'.
func normalizeFloatX80Subnormal(aSig uint64) (zExp int32, zSig uint64) {
	shiftCount := bits.LeadingZeros64(aSig)
	zSig = aSig << shiftCount
	zExp = 1 - int32(shiftCount)
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
func roundAndPackInt32(zSign bool, absZ uint64) int32 {
	roundingMode := RoundingMode
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
		Raise(ExceptionInvalid)
		if zSign {
			return math.MinInt32
		}
		return math.MaxInt32
	}
	if roundBits != 0 {
		Raise(ExceptionInexact)
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
func roundAndPackInt64(zSign bool, absZ0, absZ1 uint64) int64 {
	roundingMode := RoundingMode
	roundNearestEven := roundingMode == RoundNearestEven
	increment := int64(absZ1) < 0

	overflow := func() int64 {
		Raise(ExceptionInvalid)
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
		Raise(ExceptionInexact)
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
	if zSign {
		return math.Float64frombits(1<<63 + uint64(zExp)<<52 + zSig)
	}
	return math.Float64frombits(uint64(zExp)<<52 + zSig)
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
//     The input significand `zSig' has its binary point between bits 62
// and 61, which is 10 bits to the left of the usual location.  This shifted
// significand must be normalized or smaller.  If `zSig' is not normalized,
// `zExp' must be 0; in that case, the result returned is a subnormal number,
// and it must not require rounding.  In the usual case that `zSig' is
// normalized, `zExp' must be 1 less than the ``true'' floating-point exponent.
// The handling of underflow and overflow follows the IEC/IEEE Standard for
// Binary Floating-Point Arithmetic.
func roundAndPackFloat64(zSign bool, zExp int16, zSig uint64) float64 {
	roundingMode := RoundingMode
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
			Raise(ExceptionOverflow | ExceptionInexact)
			result := packFloat64(zSign, 0x7FF, 0)
			if roundIncrement == 0 {
				return result - 1
			}
			return result
		}
		if zExp < 0 {
			isTiny := DetectTininess == TininessBeforeRounding || zExp < -1
			zSig = shift64RightJamming(zSig, -zExp)
			zExp = 0
			roundBits = zSig & 0x3FF
			if isTiny && roundBits != 0 {
				Raise(ExceptionUnderflow)
			}
		}
	}
	if roundBits != 0 {
		Raise(ExceptionInexact)
	}
	zSig = uint64(int64(zSig) + roundIncrement>>10)
	if (roundBits^0x200) == 0 && roundNearestEven {
		zSig &= uint64(1)
	}
	if zSig == 0 {
		zExp = 0
	}
	return packFloat64(zSign, zExp, zSig)
}
