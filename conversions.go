package float

import (
	"math"
	"math/bits"
)

// Int32ToFloatX80 returns the result of converting the 32-bit two's complement integer `a'
// to the extended double-precision floating-point format.  The conversion
// is performed according to the IEC/IEEE Standard for Binary Floating-Point
// Arithmetic.
func Int32ToFloatX80(a int32) X80 {
	if a == 0 {
		return X80Zero
	}
	absA := a
	zSign := a < 0
	if zSign {
		absA = -a
	}
	shiftCount := bits.LeadingZeros32(uint32(absA)) + 32
	return packFloatX80(zSign, 0x403E-shiftCount, uint64(absA)<<shiftCount)
}

// Int64ToFloatX80 returns the result of converting the 64-bit two's complement integer `a'
// to the extended double-precision floating-point format.  The conversion
// is performed according to the IEC/IEEE Standard for Binary Floating-Point
// Arithmetic.
func Int64ToFloatX80(a int64) X80 {
	if a == 0 {
		return X80Zero
	}
	absA := a
	zSign := a < 0
	if zSign {
		absA = -a
	}
	shiftCount := bits.LeadingZeros64(uint64(absA))
	return packFloatX80(zSign, 0x403E-shiftCount, uint64(absA)<<shiftCount)
}

// Float32ToFloatX80 returns the result of converting the single-precision floating-point value
// `a' to the extended double-precision floating-point format.  The conversion
// is performed according to the IEC/IEEE Standard for Binary Floating-Point
// Arithmetic.
func Float32ToFloatX80(a float32) X80 {
	return Float64ToFloatX80(float64(a))
}

// Float64ToFloatX80 returns the result of converting the double-precision floating-point value
// `a' to the extended double-precision floating-point format.  The conversion
// is performed according to the IEC/IEEE Standard for Binary Floating-Point
// Arithmetic.
func Float64ToFloatX80(a float64) X80 {
	b := math.Float64bits(a)
	aSig := b & 0x000FFFFFFFFFFFFF
	aExp := int((b >> 52) & 0x7FF)
	aSign := b>>63 != 0
	if aExp == 0x7FF {
		if aSig != 0 {
			return X80NaN
		}
		return packFloatX80(aSign, 0x7FFF, 0x8000000000000000)
	}
	if aExp == 0 {
		if aSig == 0 {
			return packFloatX80(aSign, 0, 0)
		}
		shiftCount := bits.LeadingZeros64(aSig) - 11
		aExp, aSig = 1-shiftCount, aSig<<shiftCount
	}
	return packFloatX80(aSign, aExp+0x3C00, (aSig|0x0010000000000000)<<11)
}

// ToInt32 returns the result of converting the extended double-precision floating-
// point value `a' to the 32-bit two's complement integer format.  The
// conversion is performed according to the IEC/IEEE Standard for Binary
// Floating-Point Arithmetic---which means in particular that the conversion
// is rounded according to the current rounding mode.  If `a' is a NaN, the
// largest positive integer is returned.  Otherwise, if the conversion
// overflows, the largest integer with the same sign as `a' is returned.
func (a X80) ToInt32() int32 {
	aSig := a.frac()
	aExp := a.exp()
	aSign := a.sign()
	if (aExp == 0x7FFF) && uint64(aSig<<1) != 0 {
		aSign = false
	}
	shiftCount := 0x4037 - aExp
	if shiftCount <= 0 {
		shiftCount = 1
	}
	aSig = shift64RightJamming(aSig, int16(shiftCount))
	return roundAndPackInt32(aSign, aSig)
}

// ToInt32RoundZero returns the result of converting the extended double-precision floating-
// point value `a' to the 32-bit two's complement integer format.  The
// conversion is performed according to the IEC/IEEE Standard for Binary
// Floating-Point Arithmetic, except that the conversion is always rounded
// toward zero.  If `a' is a NaN, the largest positive integer is returned.
// Otherwise, if the conversion overflows, the largest integer with the same
// sign as `a' is returned.
func (a X80) ToInt32RoundZero() int32 {
	aSig := a.frac()
	aExp := a.exp()
	aSign := a.sign()

	invalid := func() int32 {
		Raise(ExceptionInvalid)
		if aSign {
			return math.MinInt32
		}
		return math.MaxInt32
	}

	if 0x401E < aExp {
		if aExp == 0x7FFF && uint64(aSig<<1) != 0 {
			aSign = false
		}
		return invalid()
	} else if aExp < 0x3FFF {
		if aExp != 0 || aSig != 0 {
			Raise(ExceptionInexact)
		}
		return 0
	}
	shiftCount := 0x403E - aExp
	savedASig := aSig
	aSig >>= shiftCount
	z := int32(aSig)
	if aSign {
		z = -z
	}
	if (z < 0) != aSign {
		return invalid()
	}
	if (aSig << shiftCount) != savedASig {
		Raise(ExceptionInexact)
	}
	return z
}

// ToInt64 returns the result of converting the extended double-precision floating-
// point value `a' to the 64-bit two's complement integer format.  The
// conversion is performed according to the IEC/IEEE Standard for Binary
// Floating-Point Arithmetic---which means in particular that the conversion
// is rounded according to the current rounding mode.  If `a' is a NaN,
// the largest positive integer is returned.  Otherwise, if the conversion
// overflows, the largest integer with the same sign as `a' is returned.
func (a X80) ToInt64() int64 {
	aSig := a.frac()
	aExp := a.exp()
	aSign := a.sign()
	shiftCount := 0x403E - aExp
	aSigExtra := uint64(0)
	if shiftCount < 0 {
		Raise(ExceptionInvalid)
		if !aSign || (aExp == 0x7FFF && aSig != 0x8000000000000000) {
			return math.MaxInt64
		}
		return math.MinInt64
	}
	aSig, aSigExtra = shift64ExtraRightJamming(aSig, 0, int16(shiftCount))
	return roundAndPackInt64(aSign, aSig, aSigExtra)
}

// ToInt64RoundZero returns the result of converting the extended double-precision
// floating-point value `a' to the 64-bit two's complement integer format.  The
// conversion is performed according to the IEC/IEEE Standard for Binary
// Floating-Point Arithmetic, except that the conversion is always rounded
// toward zero.  If `a' is a NaN, the largest positive integer is returned.
// Otherwise, if the conversion overflows, the largest integer with the same
// sign as `a' is returned.
func (a X80) ToInt64RoundZero() int64 {
	aSig := a.frac()
	aExp := a.exp()
	aSign := a.sign()
	shiftCount := aExp - 0x403E
	if 0 <= shiftCount {
		aSig &= math.MaxInt64
		if a.high != 0xC03E || aSig != 0 {
			Raise(ExceptionInvalid)
			if !aSign || ((aExp == 0x7FFF) && aSig != 0) {
				return math.MaxInt64
			}
		}
		return math.MaxInt64
	} else if aExp < 0x3FFF {
		if aExp != 0 || aSig != 0 {
			Raise(ExceptionInexact)
		}
		return 0
	}
	z := int64(aSig) >> (-shiftCount)
	if uint64(aSig<<(shiftCount&63)) != 0 {
		Raise(ExceptionInexact)
	}
	if aSign {
		z = -z
	}
	return z
}

// ToFloat32 returns the result of converting the extended double-precision floating-
// point value `a' to the double-precision floating-point format.  The
// conversion is performed according to the IEC/IEEE Standard for Binary
// Floating-Point Arithmetic.
func (a X80) ToFloat32() float32 {
	return float32(a.ToFloat64())
}

// ToFloat64 returns the result of converting the extended double-precision floating-
// point value `a' to the double-precision floating-point format.  The
// conversion is performed according to the IEC/IEEE Standard for Binary
// Floating-Point Arithmetic.
func (a X80) ToFloat64() float64 {
	aSig, aExp, aSign := a.frac(), a.exp(), a.sign()
	if aExp == 0x7FFF {
		if aSig<<1 != 0 {
			return math.NaN()
		}
		return packFloat64(aSign, 0x7FF, 0)
	}
	zSig := shift64RightJamming(aSig, 1)
	if aExp != 0 || aSig != 0 {
		aExp -= 0x3C01
	}
	return roundAndPackFloat64(aSign, int16(aExp), zSig)
}
