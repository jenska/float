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
// is exact; a signaling NaN raises the invalid exception and is quieted.
func (e *Env) Float32ToFloatX80(a float32) X80 {
	return e.NewFromFloat32Bits(math.Float32bits(a))
}

// Float64ToFloatX80 returns the result of converting the double-precision floating-point value
// `a' to the extended double-precision floating-point format.  The conversion
// is exact; a signaling NaN raises the invalid exception and is quieted.
func (e *Env) Float64ToFloatX80(a float64) X80 {
	return e.NewFromFloat64Bits(math.Float64bits(a))
}

// NewFromFloat32Bits converts the IEEE 754 single-precision value with bit
// pattern b to the extended format.  Unlike Float32ToFloatX80 with a float32
// argument, the bit pattern cannot be altered by Go's float handling, so the
// payload and signaling state of a NaN are preserved: a signaling NaN raises
// the invalid exception and becomes the quiet NaN with the same payload.
func (e *Env) NewFromFloat32Bits(b uint32) X80 {
	aSig := b & 0x007FFFFF
	aExp := int((b >> 23) & 0xFF)
	aSign := b>>31 != 0
	if aExp == 0xFF {
		if aSig != 0 {
			if aSig&0x00400000 == 0 {
				e.Raise(ExceptionInvalid)
			}
			return packFloatX80(aSign, 0x7FFF, 0xC000000000000000|uint64(aSig)<<40)
		}
		return packFloatX80(aSign, 0x7FFF, 0x8000000000000000)
	}
	if aExp == 0 {
		if aSig == 0 {
			return packFloatX80(aSign, 0, 0)
		}
		shiftCount := bits.LeadingZeros32(aSig) - 8
		aExp, aSig = 1-shiftCount, aSig<<shiftCount
	}
	return packFloatX80(aSign, aExp+0x3F80, uint64(aSig|0x00800000)<<40)
}

// NewFromFloat64Bits converts the IEEE 754 double-precision value with bit
// pattern b to the extended format, preserving NaN payloads like
// NewFromFloat32Bits.
func (e *Env) NewFromFloat64Bits(b uint64) X80 {
	aSig := b & 0x000FFFFFFFFFFFFF
	aExp := int((b >> 52) & 0x7FF)
	aSign := b>>63 != 0
	if aExp == 0x7FF {
		if aSig != 0 {
			if aSig&0x0008000000000000 == 0 {
				e.Raise(ExceptionInvalid)
			}
			return packFloatX80(aSign, 0x7FFF, 0xC000000000000000|aSig<<11)
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
func (e *Env) ToInt32(a X80) int32 {
	a = a.canonical()
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
	return e.roundAndPackInt32(aSign, aSig)
}

// ToInt32RoundZero returns the result of converting the extended double-precision floating-
// point value `a' to the 32-bit two's complement integer format.  The
// conversion is performed according to the IEC/IEEE Standard for Binary
// Floating-Point Arithmetic, except that the conversion is always rounded
// toward zero.  If `a' is a NaN, the largest positive integer is returned.
// Otherwise, if the conversion overflows, the largest integer with the same
// sign as `a' is returned.
func (e *Env) ToInt32RoundZero(a X80) int32 {
	a = a.canonical()
	aSig := a.frac()
	aExp := a.exp()
	aSign := a.sign()

	invalid := func() int32 {
		e.Raise(ExceptionInvalid)
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
			e.Raise(ExceptionInexact)
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
		e.Raise(ExceptionInexact)
	}
	return z
}

// ToInt16 returns the result of converting `a' to a 16-bit two's complement
// integer, rounded according to the current rounding mode.  If `a' is a NaN
// the largest positive integer is returned; if the conversion overflows the
// largest integer with the sign of `a' is returned.  Both raise the invalid
// exception.
func (e *Env) ToInt16(a X80) int16 {
	return int16(e.toIntN(a, math.MinInt16, math.MaxInt16))
}

// ToInt8 returns the result of converting `a' to an 8-bit two's complement
// integer, with the same rounding and overflow behavior as ToInt16.
func (e *Env) ToInt8(a X80) int8 {
	return int8(e.toIntN(a, math.MinInt8, math.MaxInt8))
}

func (e *Env) toIntN(a X80, lo, hi int32) int32 {
	q := e.quiet()
	z := q.ToInt32(a)
	raised := q.Exception
	if raised&ExceptionInvalid != 0 || z < lo || z > hi {
		e.Raise(ExceptionInvalid)
		if z < 0 {
			return lo
		}
		return hi
	}
	if raised != 0 {
		e.Raise(raised)
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
func (e *Env) ToInt64(a X80) int64 {
	a = a.canonical()
	aSig := a.frac()
	aExp := a.exp()
	aSign := a.sign()
	shiftCount := 0x403E - aExp
	aSigExtra := uint64(0)
	if shiftCount < 0 {
		e.Raise(ExceptionInvalid)
		if !aSign || (aExp == 0x7FFF && aSig != 0x8000000000000000) {
			return math.MaxInt64
		}
		return math.MinInt64
	}
	aSig, aSigExtra = shift64ExtraRightJamming(aSig, 0, int16(shiftCount))
	return e.roundAndPackInt64(aSign, aSig, aSigExtra)
}

// ToInt64RoundZero returns the result of converting the extended double-precision
// floating-point value `a' to the 64-bit two's complement integer format.  The
// conversion is performed according to the IEC/IEEE Standard for Binary
// Floating-Point Arithmetic, except that the conversion is always rounded
// toward zero.  If `a' is a NaN, the largest positive integer is returned.
// Otherwise, if the conversion overflows, the largest integer with the same
// sign as `a' is returned.
func (e *Env) ToInt64RoundZero(a X80) int64 {
	a = a.canonical()
	aSig := a.frac()
	aExp := a.exp()
	aSign := a.sign()
	shiftCount := aExp - 0x403E
	if 0 <= shiftCount {
		aSig &= math.MaxInt64
		if a.high != 0xC03E || aSig != 0 {
			e.Raise(ExceptionInvalid)
			if !aSign || ((aExp == 0x7FFF) && aSig != 0) {
				return math.MaxInt64
			}
		}
		return math.MinInt64
	} else if aExp < 0x3FFF {
		if aExp != 0 || aSig != 0 {
			e.Raise(ExceptionInexact)
		}
		return 0
	}
	z := int64(aSig >> -shiftCount)
	if uint64(aSig<<(shiftCount&63)) != 0 {
		e.Raise(ExceptionInexact)
	}
	if aSign {
		z = -z
	}
	return z
}

// ToFloat32 returns the result of converting the extended double-precision floating-
// point value `a' to the single-precision floating-point format.  The
// conversion is performed according to the IEC/IEEE Standard for Binary
// Floating-Point Arithmetic.
func (e *Env) ToFloat32(a X80) float32 {
	return math.Float32frombits(e.ToFloat32Bits(a))
}

// ToFloat32Bits is like ToFloat32 but returns the IEEE 754 bit pattern, which
// preserves NaN payloads: a signaling NaN raises the invalid exception and
// becomes the quiet NaN with the top bits of the payload.
func (e *Env) ToFloat32Bits(a X80) uint32 {
	a = a.canonical()
	aSig, aExp, aSign := a.frac(), a.exp(), a.sign()
	if aExp == 0x7FFF {
		if aSig<<1 != 0 {
			if a.IsSignalingNaN() {
				e.Raise(ExceptionInvalid)
			}
			return uint32(x1(aSign))<<31 | 0x7FC00000 | uint32(aSig<<1>>41)
		}
		return math.Float32bits(packFloat32(aSign, 0xFF, 0))
	}
	zSig := uint32(shift64RightJamming(aSig, 33))
	if aExp != 0 || aSig != 0 {
		aExp -= 0x3F81
	}
	return math.Float32bits(e.roundAndPackFloat32(aSign, int16(aExp), zSig))
}

// ToFloat64 returns the result of converting the extended double-precision floating-
// point value `a' to the double-precision floating-point format.  The
// conversion is performed according to the IEC/IEEE Standard for Binary
// Floating-Point Arithmetic.
func (e *Env) ToFloat64(a X80) float64 {
	return math.Float64frombits(e.ToFloat64Bits(a))
}

// ToFloat64Bits is like ToFloat64 but returns the IEEE 754 bit pattern,
// preserving NaN payloads like ToFloat32Bits.
func (e *Env) ToFloat64Bits(a X80) uint64 {
	a = a.canonical()
	aSig, aExp, aSign := a.frac(), a.exp(), a.sign()
	if aExp == 0x7FFF {
		if aSig<<1 != 0 {
			if a.IsSignalingNaN() {
				e.Raise(ExceptionInvalid)
			}
			return x1(aSign)<<63 | 0x7FF8000000000000 | aSig<<1>>12
		}
		return math.Float64bits(packFloat64(aSign, 0x7FF, 0))
	}
	zSig := shift64RightJamming(aSig, 1)
	if aExp != 0 || aSig != 0 {
		aExp -= 0x3C01
	}
	return math.Float64bits(e.roundAndPackFloat64(aSign, int16(aExp), zSig))
}
