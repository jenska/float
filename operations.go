package float

// RoundToInt rounds the extended double-precision floating-point value `a' to an integer,
// and returns the result as an extended quadruple-precision floating-point
// value.  The operation is performed according to the IEC/IEEE Standard for
// Binary Floating-Point Arithmetic.
func (e *Env) RoundToInt(a X80) X80 {
	return e.roundToInt(a, e.RoundingMode)
}

func (e *Env) roundToInt(a X80, roundingMode int) X80 {
	a = a.canonical()
	aExp := a.exp()
	if 0x403E <= aExp {
		if aExp == 0x7FFF && a.frac()<<1 != 0 {
			return e.propagateFloatX80NaN(a, a)
		}
		return a
	}
	if aExp < 0x3FFF {
		if aExp == 0 && a.frac()<<1 == 0 {
			return a
		}
		e.Raise(ExceptionInexact)
		aSign := a.sign()
		switch roundingMode {
		case RoundNearestEven:
			if aExp == 0x3FFE && a.frac()<<1 != 0 {
				return packFloatX80(aSign, 0x3FFF, 0x8000000000000000)
			}
		case RoundDown:
			if aSign {
				return packFloatX80(true, 0x3FFF, 0x8000000000000000)
			}
			return X80Zero
		case RoundUp:
			if aSign {
				return packFloatX80(true, 0, 0)
			}
			return packFloatX80(false, 0x3FFF, 0x8000000000000000)
		}
		return packFloatX80(aSign, 0, 0)
	}
	lastBitMask := uint64(1 << (0x403E - aExp))
	roundBitsMask := lastBitMask - 1
	z := a
	if roundingMode == RoundNearestEven {
		z.low += lastBitMask >> 1
		if z.low&roundBitsMask == 0 {
			z.low &= ^lastBitMask
		}
	} else if roundingMode != RoundToZero {
		if z.sign() != (roundingMode == RoundUp) {
			z.low += roundBitsMask
		}
	}
	z.low &= ^roundBitsMask
	if z.low == 0 {
		z.high++
		z.low = 0x8000000000000000
	}
	if z.low != a.low {
		e.Raise(ExceptionInexact)
	}
	return z
}

// Add returns the result of adding the extended double-precision floating-point
// values `a' and `b'.  The operation is performed according to the IEC/IEEE
// Standard for Binary Floating-Point Arithmetic.
func (e *Env) Add(a X80, b X80) X80 {
	a, b = a.canonical(), b.canonical()
	aSign, bSign := a.sign(), b.sign()
	if aSign == bSign {
		return e.addFloatx80Sigs(a, b, aSign)
	}
	return e.subFloatx80Sigs(a, b, aSign)
}

// Sub returns the result of subtracting the extended double-precision floating-
// point values `a' and `b'.  The operation is performed according to the
// IEC/IEEE Standard for Binary Floating-Point Arithmetic.
func (e *Env) Sub(a X80, b X80) X80 {
	a, b = a.canonical(), b.canonical()
	aSign, bSign := a.sign(), b.sign()
	if aSign == bSign {
		return e.subFloatx80Sigs(a, b, aSign)
	}
	return e.addFloatx80Sigs(a, b, aSign)

}

// Returns the result of adding the absolute values of the extended double-
// precision floating-point values `a' and `b'.  If `zSign' is 1, the sum is
// negated before being returned.  `zSign' is ignored if the result is a NaN.
// The addition is performed according to the IEC/IEEE Standard for Binary
// Floating-Point Arithmetic.
func (e *Env) addFloatx80Sigs(a, b X80, zSign bool) X80 {
	aSig, bSig := a.frac(), b.frac()
	aExp, bExp := a.exp(), b.exp()
	var zSig0, zSig1 uint64
	var zExp int
	expDiff := aExp - bExp
	if 0 < expDiff {
		if aExp == 0x7FFF {
			if aSig<<1 != 0 {
				return e.propagateFloatX80NaN(a, b)
			}
			return a
		}
		if bExp == 0 {
			expDiff--
		}
		bSig, zSig1 = shift64ExtraRightJamming(bSig, 0, int16(expDiff))
		zExp = aExp
	} else if expDiff < 0 {
		if bExp == 0x7FFF {
			if bSig<<1 != 0 {
				return e.propagateFloatX80NaN(a, b)
			}
			return packFloatX80(zSign, 0x7FFF, 0x8000000000000000)
		}
		if aExp == 0 {
			expDiff++
		}
		aSig, zSig1 = shift64ExtraRightJamming(aSig, 0, int16(-expDiff))
		zExp = bExp
	} else {
		if aExp == 0x7FFF {
			if (aSig|bSig)<<1 != 0 {
				return e.propagateFloatX80NaN(a, b)
			}
			return a
		}
		zSig1 = 0
		zSig0 = aSig + bSig
		if aExp == 0 {
			zExp, zSig0 = normalizeFloatX80Subnormal(zSig0)
			return e.roundAndPackFloatX80(e.RoundingPrecision, zSign, zExp, zSig0, zSig1)
		}
		zExp = aExp
		goto shiftRight
	}
	zSig0 = aSig + bSig
	if int64(zSig0) < 0 {
		return e.roundAndPackFloatX80(e.RoundingPrecision, zSign, zExp, zSig0, zSig1)
	}
shiftRight:
	zSig0, zSig1 = shift64ExtraRightJamming(zSig0, zSig1, 1)
	zSig0 |= 0x8000000000000000
	zExp++
	return e.roundAndPackFloatX80(e.RoundingPrecision, zSign, zExp, zSig0, zSig1)
}

// Returns the result of subtracting the absolute values of the extended
// double-precision floating-point values `a' and `b'.  If `zSign' is 1, the
// difference is negated before being returned.  `zSign' is ignored if the
// result is a NaN.  The subtraction is performed according to the IEC/IEEE
// Standard for Binary Floating-Point Arithmetic.
func (e *Env) subFloatx80Sigs(a, b X80, zSign bool) X80 {
	aSig, bSig := a.frac(), b.frac()
	aExp, bExp := a.exp(), b.exp()
	var zSig0, zSig1 uint64
	var zExp int
	expDiff := aExp - bExp

	if 0 < expDiff {
		goto aExpBigger
	}
	if expDiff < 0 {
		goto bExpBigger
	}
	if aExp == 0x7FFF {
		if (aSig|bSig)<<1 != 0 {
			return e.propagateFloatX80NaN(a, b)
		}
		e.Raise(ExceptionInvalid)
		return e.defaultNaN()
	}
	if aExp == 0 {
		aExp, bExp = 1, 1
	}
	zSig1 = 0
	if bSig < aSig {
		goto aBigger
	}
	if aSig < bSig {
		goto bBigger
	}
	return packFloatX80(e.RoundingMode == RoundDown, 0, 0)
bExpBigger:
	if bExp == 0x7FFF {
		if bSig<<1 != 0 {
			return e.propagateFloatX80NaN(a, b)
		}
		return packFloatX80(!zSign, 0x7FFF, 0x8000000000000000)
	}
	if aExp == 0 {
		expDiff++
	}
	aSig, zSig1 = shift128RightJamming(aSig, 0, int16(-expDiff))
bBigger:
	zSig0, zSig1 = sub128(bSig, 0, aSig, zSig1)
	zExp = bExp
	zSign = !zSign
	goto normalizeRoundAndPack
aExpBigger:
	if aExp == 0x7FFF {
		if uint64(aSig<<1) != 0 {
			return e.propagateFloatX80NaN(a, b)
		}
		return a
	}
	if bExp == 0 {
		expDiff--
	}
	bSig, zSig1 = shift128RightJamming(bSig, 0, int16(expDiff))
aBigger:
	zSig0, zSig1 = sub128(aSig, 0, bSig, zSig1)
	zExp = aExp
normalizeRoundAndPack:
	return e.normalizeRoundAndPackFloatX80(
		e.RoundingPrecision, zSign, zExp, zSig0, zSig1)

}

// Mul returns the result of multiplying the extended double-precision floating-
// point values `a' and `b'.  The operation is performed according to the
// IEC/IEEE Standard for Binary Floating-Point Arithmetic.
func (e *Env) Mul(a X80, b X80) X80 {
	return e.mulFloatX80(a.canonical(), b.canonical(), e.RoundingPrecision, false)
}

// SglMul returns the product of `a' and `b' in single precision: the
// significands of both operands are truncated to 24 bits and the product is
// rounded to 24 bits, while the exponent keeps the extended range.  The
// RoundingPrecision setting is ignored.  This is the 68881/68882 FSGLMUL
// operation.
func (e *Env) SglMul(a X80, b X80) X80 {
	return e.mulFloatX80(a.canonical(), b.canonical(), 32, true)
}

func (e *Env) mulFloatX80(a, b X80, prec int, sgl bool) X80 {
	aSig, aExp, aSign := a.frac(), a.exp(), a.sign()
	bSig, bExp, bSign := b.frac(), b.exp(), b.sign()
	zSign := aSign != bSign

	if aExp == 0x7FFF {
		if aSig<<1 != 0 || (bExp == 0x7FFF && bSig<<1 != 0) {
			return e.propagateFloatX80NaN(a, b)
		}
		if bExp == 0 && bSig == 0 {
			e.Raise(ExceptionInvalid)
			return e.defaultNaN()
		}
		return packFloatX80(zSign, 0x7FFF, 0x8000000000000000)
	}

	if bExp == 0x7FFF {
		if bSig<<1 != 0 {
			return e.propagateFloatX80NaN(a, b)
		}
		if aExp == 0 && aSig == 0 {
			e.Raise(ExceptionInvalid)
			return e.defaultNaN()
		}
		return packFloatX80(zSign, 0x7FFF, 0x8000000000000000)
	}
	if aExp == 0 {
		if aSig == 0 {
			return packFloatX80(zSign, 0, 0)
		}
		aExp, aSig = normalizeFloatX80Subnormal(aSig)
	}
	if bExp == 0 {
		if bSig == 0 {
			return packFloatX80(zSign, 0, 0)
		}
		bExp, bSig = normalizeFloatX80Subnormal(bSig)
	}
	if sgl {
		aSig &= 0xFFFFFF0000000000
		bSig &= 0xFFFFFF0000000000
	}
	zExp := aExp + bExp - 0x3FFE
	zSig0, zSig1 := mul64To128(aSig, bSig)
	if int64(zSig0) > 0 {
		zSig0, zSig1 = shortShift128Left(zSig0, zSig1, 1)
		zExp--
	}
	return e.roundAndPackFloatX80(prec, zSign, zExp, zSig0, zSig1)
}

// Div returns the result of dividing the extended double-precision floating-point
// value `a' by the corresponding value `b'.  The operation is performed
// according to the IEC/IEEE Standard for Binary Floating-Point Arithmetic.
func (e *Env) Div(a X80, b X80) X80 {
	return e.divFloatX80(a.canonical(), b.canonical(), e.RoundingPrecision)
}

// SglDiv returns the quotient of `a' and `b' rounded to single precision,
// while the exponent keeps the extended range.  The RoundingPrecision setting
// is ignored.  This is the 68881/68882 FSGLDIV operation.
func (e *Env) SglDiv(a X80, b X80) X80 {
	return e.divFloatX80(a.canonical(), b.canonical(), 32)
}

func (e *Env) divFloatX80(a, b X80, prec int) X80 {
	aSig, aExp, aSign := a.frac(), a.exp(), a.sign()
	bSig, bExp, bSign := b.frac(), b.exp(), b.sign()
	zSign := aSign != bSign
	if aExp == 0x7FFF {
		if uint64(aSig<<1) != 0 {
			return e.propagateFloatX80NaN(a, b)
		}
		if bExp == 0x7FFF {
			if uint64(bSig<<1) != 0 {
				return e.propagateFloatX80NaN(a, b)
			}
			e.Raise(ExceptionInvalid)
			return e.defaultNaN()
		}
		return packFloatX80(zSign, 0x7FFF, 0x8000000000000000)
	}
	if bExp == 0x7FFF {
		if bSig<<1 != 0 {
			return e.propagateFloatX80NaN(a, b)
		}
		return packFloatX80(zSign, 0, 0)
	}
	if bExp == 0 {
		if bSig == 0 {
			if aExp == 0 && aSig == 0 {
				e.Raise(ExceptionInvalid)
				return e.defaultNaN()
			}
			e.Raise(ExceptionDivbyzero)
			return packFloatX80(zSign, 0x7FFF, 0x8000000000000000)
		}
		bExp, bSig = normalizeFloatX80Subnormal(bSig)
	}
	if aExp == 0 {
		if aSig == 0 {
			return packFloatX80(zSign, 0, 0)
		}
		aExp, aSig = normalizeFloatX80Subnormal(aSig)
	}
	zExp := aExp - bExp + 0x3FFE
	var rem0, rem1, rem2, term2 uint64
	if bSig <= aSig {
		aSig, rem1 = shift128Right(aSig, 0, 1)
		zExp++
	}
	zSig0 := estimateDiv128To64(aSig, rem1, bSig)
	term0, term1 := mul64To128(bSig, zSig0)
	rem0, rem1 = sub128(aSig, rem1, term0, term1)
	for int64(rem0) < 0 {
		zSig0--
		rem0, rem1 = add128(rem0, rem1, 0, bSig)
	}
	zSig1 := estimateDiv128To64(rem1, 0, bSig)
	if zSig1<<1 <= 8 {
		term1, term2 = mul64To128(bSig, zSig1)
		rem1, rem2 = sub128(rem1, 0, term1, term2)
		for int64(rem1) < 0 {
			zSig1--
			rem1, rem2 = add128(rem1, rem2, 0, bSig)
		}
		if rem1|rem2 != 0 {
			zSig1 |= 1
		}
	}
	return e.roundAndPackFloatX80(prec, zSign, zExp, zSig0, zSig1)
}

// Rem returns the IEEE remainder of `a' with respect to `b': a - n*b where n
// is the integer nearest to a/b, ties to even.  The result is exact.  This is
// the 68881/68882 FREM operation.
func (e *Env) Rem(a X80, b X80) X80 {
	z, _ := e.remFloatX80(a, b, false)
	return z
}

// RemQuo returns Rem(b) together with the low-order bits of the quotient n:
// quo has the sign of a/b and its magnitude is congruent to |n| modulo 2^31.
// The 68881/68882 quotient byte is the sign and the low 7 bits of quo.
func (e *Env) RemQuo(a X80, b X80) (z X80, quo int) {
	return e.remFloatX80(a, b, false)
}

// Mod returns the truncated remainder of `a' with respect to `b': a - n*b
// where n is a/b rounded toward zero.  The result has the sign of `a' and is
// exact.  This is the C fmod function and the 68881/68882 FMOD operation.
func (e *Env) Mod(a X80, b X80) X80 {
	z, _ := e.ModQuo(a, b)
	return z
}

// ModQuo returns Mod(b) together with the low-order bits of the truncated
// quotient, in the same form as RemQuo.
func (e *Env) ModQuo(a X80, b X80) (z X80, quo int) {
	return e.remFloatX80(a, b, true)
}

// remFloatX80 computes the remainder of `a' with respect to `b' and the low
// 64 bits of the quotient, which is rounded to nearest-even, or toward zero
// if `truncate' is set.
func (e *Env) remFloatX80(a, b X80, truncate bool) (X80, int) {
	a, b = a.canonical(), b.canonical()
	aSig0, aExp, aSign := a.frac(), a.exp(), a.sign()
	bSig, bExp, bSign := b.frac(), b.exp(), b.sign()
	var term0, term1, q uint64

	if aExp == 0x7FFF {
		if aSig0<<1 != 0 || (bExp == 0x7FFF && bSig<<1 != 0) {
			return e.propagateFloatX80NaN(a, b), 0
		}
		e.Raise(ExceptionInvalid)
		return e.defaultNaN(), 0
	}
	if bExp == 0x7FFF {
		if bSig<<1 != 0 {
			return e.propagateFloatX80NaN(a, b), 0
		}
		return a, 0
	}
	if bExp == 0 {
		if bSig == 0 {
			e.Raise(ExceptionInvalid)
			return e.defaultNaN(), 0
		}
		bExp, bSig = normalizeFloatX80Subnormal(bSig)
	}
	if aExp == 0 {
		if aSig0 == 0 {
			return a, 0
		}
		aExp, aSig0 = normalizeFloatX80Subnormal(aSig0)
	}
	zSign := aSign
	expDiff := aExp - bExp
	aSig1 := uint64(0)
	if expDiff < 0 {
		if expDiff < -1 {
			return a, 0
		}
		aSig0, aSig1 = shift128Right(aSig0, 0, 1)
		expDiff = 0
	}
	// The quotient is assembled from partial quotients: the first is one bit,
	// the loop yields 64 and then 62 new bits per step, the last step expDiff.
	q = x1(bSig <= aSig0)
	if q != 0 {
		aSig0 -= bSig
	}
	quo := q
	shift := 64
	expDiff -= 64
	for 0 < expDiff {
		q = estimateDiv128To64(aSig0, aSig1, bSig)
		if 2 < q {
			q -= 2
		} else {
			q = 0
		}
		quo = quo<<shift + q
		shift = 62
		term0, term1 = mul64To128(bSig, q)
		aSig0, aSig1 = sub128(aSig0, aSig1, term0, term1)
		aSig0, aSig1 = shortShift128Left(aSig0, aSig1, 62)
		expDiff -= 62
	}
	expDiff += 64
	if 0 < expDiff {
		q = estimateDiv128To64(aSig0, aSig1, bSig)
		if 2 < q {
			q -= 2
		} else {
			q = 0
		}
		q >>= 64 - expDiff
		term0, term1 = mul64To128(bSig, q<<(64-expDiff))
		aSig0, aSig1 = sub128(aSig0, aSig1, term0, term1)
		term0, term1 = shortShift128Left(0, bSig, int16(64-expDiff))
		for le128(term0, term1, aSig0, aSig1) {
			q++
			aSig0, aSig1 = sub128(aSig0, aSig1, term0, term1)
		}
		quo = quo<<(expDiff-64+shift) + q
	} else {
		term1 = 0
		term0 = bSig
	}
	if !truncate {
		alternateASig0, alternateASig1 := sub128(term0, term1, aSig0, aSig1)
		if lt128(alternateASig0, alternateASig1, aSig0, aSig1) ||
			eq128(alternateASig0, alternateASig1, aSig0, aSig1) &&
				(q&1) != 0 {
			aSig0 = alternateASig0
			aSig1 = alternateASig1
			zSign = !zSign
			quo++
		}
	}
	n := int(quo & 0x7FFFFFFF)
	if aSign != bSign {
		n = -n
	}
	return e.normalizeRoundAndPackFloatX80(80, zSign, bExp+expDiff, aSig0, aSig1), n
}

// Scale returns `a' * 2^n, rounded according to the current rounding mode and
// precision, with overflow and underflow handled as for any other operation.
// This is the C scalbn function and the 68881/68882 FSCALE operation.
func (e *Env) Scale(a X80, n int) X80 {
	a = a.canonical()
	aSig, aExp, aSign := a.frac(), a.exp(), a.sign()
	if aExp == 0x7FFF {
		if aSig<<1 != 0 {
			return e.propagateFloatX80NaN(a, a)
		}
		return a
	}
	if aExp == 0 {
		if aSig == 0 {
			return a
		}
		aExp, aSig = normalizeFloatX80Subnormal(aSig)
	}
	// beyond these limits the result over- or underflows anyway; the clamp
	// keeps the exponent within what roundAndPackFloatX80 can shift
	zExp := aExp + max(min(n, 0x10000), -0x10000)
	zExp = max(zExp, -0x7000)
	return e.roundAndPackFloatX80(e.RoundingPrecision, aSign, zExp, aSig, 0)
}

// GetExp returns the unbiased binary exponent of `a' as an X80 value, so that
// a = GetMan(a) * 2^GetExp(a) for finite nonzero `a'.  Subnormal values are
// normalized first.  GetExp(±0) = ±0; GetExp(±Inf) raises the invalid
// exception and returns DefaultNaN.  This is the 68881/68882 FGETEXP operation.
func (e *Env) GetExp(a X80) X80 {
	a = a.canonical()
	aSig, aExp := a.frac(), a.exp()
	if aExp == 0x7FFF {
		if aSig<<1 != 0 {
			return e.propagateFloatX80NaN(a, a)
		}
		e.Raise(ExceptionInvalid)
		return e.defaultNaN()
	}
	if aExp == 0 {
		if aSig == 0 {
			return a
		}
		aExp, _ = normalizeFloatX80Subnormal(aSig)
	}
	return Int32ToFloatX80(int32(aExp - 0x3FFF))
}

// GetMan returns the significand of `a' scaled to 1 <= |m| < 2, with the sign
// of `a'.  Subnormal values are normalized first.  GetMan(±0) = ±0;
// GetMan(±Inf) raises the invalid exception and returns DefaultNaN.  This is
// the 68881/68882 FGETMAN operation.
func (e *Env) GetMan(a X80) X80 {
	a = a.canonical()
	aSig, aExp, aSign := a.frac(), a.exp(), a.sign()
	if aExp == 0x7FFF {
		if aSig<<1 != 0 {
			return e.propagateFloatX80NaN(a, a)
		}
		e.Raise(ExceptionInvalid)
		return e.defaultNaN()
	}
	if aExp == 0 {
		if aSig == 0 {
			return a
		}
		_, aSig = normalizeFloatX80Subnormal(aSig)
	}
	return packFloatX80(aSign, 0x3FFF, aSig)
}

// RoundToPrecision rounds `a' to `prec' significand bits, which is 32 (24
// bits, single), 64 (53 bits, double) or 80 (64 bits, full extended), using
// the current rounding mode.  The exponent keeps the extended range.  It is
// how a value is brought to the precision selected in the 68881/68882 FPCR.
func (e *Env) RoundToPrecision(a X80, prec int) X80 {
	a = a.canonical()
	aSig, aExp, aSign := a.frac(), a.exp(), a.sign()
	if aExp == 0x7FFF {
		if aSig<<1 != 0 {
			return e.propagateFloatX80NaN(a, a)
		}
		return a
	}
	if aSig == 0 {
		return a
	}
	if aExp == 0 {
		aExp, aSig = normalizeFloatX80Subnormal(aSig)
	}
	return e.roundAndPackFloatX80(prec, aSign, aExp, aSig, 0)
}

// Trunc rounds `a' to an integer toward zero, regardless of the current
// rounding mode.  This is the 68881/68882 FINTRZ operation; RoundToInt is FINT.
func (e *Env) Trunc(a X80) X80 {
	return e.roundToInt(a, RoundToZero)
}

// Sqrt returns the square root of the extended double-precision floating-point
// value `a'.  The operation is performed according to the IEC/IEEE Standard
// for Binary Floating-Point Arithmetic.
func (e *Env) Sqrt(a X80) X80 {
	a = a.canonical()
	aSig0, aExp, aSign := a.frac(), a.exp(), a.sign()
	var aSig1 uint64
	if aExp == 0x7FFF {
		if aSig0<<1 != 0 {
			return e.propagateFloatX80NaN(a, a)
		}
		if !aSign {
			return a
		}
		e.Raise(ExceptionInvalid)
		return e.defaultNaN()
	}
	if aSign {
		if aExp == 0 && aSig0 == 0 {
			return a
		}
		e.Raise(ExceptionInvalid)
		return e.defaultNaN()
	}
	if aExp == 0 {
		if aSig0 == 0 {
			return X80Zero
		}
		aExp, aSig0 = normalizeFloatX80Subnormal(aSig0)
	}
	zExp := ((aExp - 0x3FFF) >> 1) + 0x3FFF
	zSig0 := uint64(estimateSqrt32(int32(aExp), uint32(aSig0>>32)))
	aSig0, aSig1 = shift128Right(aSig0, 0, int16(2+(aExp&1)))
	zSig0 = estimateDiv128To64(aSig0, aSig1, zSig0<<32) + (zSig0 << 30)
	doubleZSig0 := zSig0 << 1
	term0, term1 := mul64To128(zSig0, zSig0)
	rem0, rem1 := sub128(aSig0, aSig1, term0, term1)
	for int64(rem0) < 0 {
		zSig0--
		doubleZSig0 -= 2
		rem0, rem1 = add128(rem0, rem1, zSig0>>63, doubleZSig0|1)
	}
	zSig1 := estimateDiv128To64(rem1, 0, doubleZSig0)
	if (zSig1 & 0x3FFFFFFFFFFFFFFF) <= 5 {
		if zSig1 == 0 {
			zSig1 = 1
		}
		term1, term2 := mul64To128(doubleZSig0, zSig1)
		rem1, rem2 := sub128(rem1, 0, term1, term2)
		term2, term3 := mul64To128(zSig1, zSig1)
		rem1, rem2, rem3 := sub192(rem1, rem2, 0, 0, term2, term3)
		for int(rem1) < 0 {
			zSig1--
			term2, term3 = shortShift128Left(0, zSig1, 1)
			term3 |= 1
			term2 |= doubleZSig0
			rem1, rem2, rem3 = add192(rem1, rem2, rem3, 0, term2, term3)
		}
		if (rem1 | rem2 | rem3) != 0 {
			zSig1 |= uint64(1)
		}
	}
	zSig0, zSig1 = shortShift128Left(0, zSig1, 1)
	zSig0 |= doubleZSig0
	return e.roundAndPackFloatX80(e.RoundingPrecision, false, zExp, zSig0, zSig1)
}
