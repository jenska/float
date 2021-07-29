package float

// Rounds the extended double-precision floating-point value `a' to an integer,
// and returns the result as an extended quadruple-precision floating-point
//value.  The operation is performed according to the IEC/IEEE Standard for
// Binary Floating-Point Arithmetic.
func (a X80) RoundToInt() X80 {
	aExp := a.exp()
	if 0x403E <= aExp {
		if aExp == 0x7FFF && a.frac()<<1 != 0 {
			return propagateFloatX80NaN(a, a)
		}
		return a
	}
	if aExp < 0x3FFF {
		if aExp == 0 && a.frac()<<1 == 0 {
			return a
		}
		Raise(ExceptionInexact)
		aSign := a.sign()
		switch RoundingMode {
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
	roundingMode := RoundingMode
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
		Raise(ExceptionInexact)
	}
	return z
}

// Returns the result of adding the extended double-precision floating-point
// values `a' and `b'.  The operation is performed according to the IEC/IEEE
// Standard for Binary Floating-Point Arithmetic.
func (a X80) Add(b X80) X80 {
	if aSign, bSign := a.sign(), b.sign(); aSign == bSign {
		return addFloatx80Sigs(a, b, aSign)
	} else {
		return subFloatx80Sigs(a, b, aSign)
	}
}

// Returns the result of subtracting the extended double-precision floating-
// point values `a' and `b'.  The operation is performed according to the
// IEC/IEEE Standard for Binary Floating-Point Arithmetic.
func (a X80) Sub(b X80) X80 {
	if aSign, bSign := a.sign(), b.sign(); aSign == bSign {
		return subFloatx80Sigs(a, b, aSign)
	} else {
		return addFloatx80Sigs(a, b, aSign)
	}
}

// Returns the result of adding the absolute values of the extended double-
// precision floating-point values `a' and `b'.  If `zSign' is 1, the sum is
// negated before being returned.  `zSign' is ignored if the result is a NaN.
// The addition is performed according to the IEC/IEEE Standard for Binary
// Floating-Point Arithmetic.
func addFloatx80Sigs(a, b X80, zSign bool) X80 {
	aSig, bSig := a.frac(), b.frac()
	aExp, bExp := a.exp(), b.exp()
	var zSig0, zSig1 uint64
	var zExp int32
	expDiff := aExp - bExp
	if 0 < expDiff {
		if aExp == 0x7FFF {
			if aSig<<1 != 0 {
				return propagateFloatX80NaN(a, b)
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
				return propagateFloatX80NaN(a, b)
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
				return propagateFloatX80NaN(a, b)
			}
			return a
		}
		zSig1 = 0
		zSig0 = aSig + bSig
		if aExp == 0 {
			zExp, zSig0 = normalizeFloatX80Subnormal(zSig0)
			return roundAndPackFloatX80(RoundingPrecision, zSign, zExp, zSig0, zSig1)
		}
		zExp = aExp
		goto shiftRight
	}
	zSig0 = aSig + bSig
	if int64(zSig0) < 0 {
		return roundAndPackFloatX80(RoundingPrecision, zSign, zExp, zSig0, zSig1)
	}
shiftRight:
	zSig0, zSig1 = shift64ExtraRightJamming(zSig0, zSig1, 1)
	zSig0 |= 0x8000000000000000
	zExp++
	return roundAndPackFloatX80(RoundingPrecision, zSign, zExp, zSig0, zSig1)
}

// Returns the result of subtracting the absolute values of the extended
// double-precision floating-point values `a' and `b'.  If `zSign' is 1, the
// difference is negated before being returned.  `zSign' is ignored if the
// result is a NaN.  The subtraction is performed according to the IEC/IEEE
// Standard for Binary Floating-Point Arithmetic.
func subFloatx80Sigs(a, b X80, zSign bool) X80 {
	aSig, bSig := a.frac(), b.frac()
	aExp, bExp := a.exp(), b.exp()
	var zSig0, zSig1 uint64
	var zExp int32
	expDiff := aExp - bExp

	if 0 < expDiff {
		goto aExpBigger
	}
	if expDiff < 0 {
		goto bExpBigger
	}
	if aExp == 0x7FFF {
		if (aSig|bSig)<<1 != 0 {
			return propagateFloatX80NaN(a, b)
		}
		Raise(ExceptionInvalid)
		return X80NaN
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
	return packFloatX80(RoundingMode == RoundDown, 0, 0)
bExpBigger:
	if bExp == 0x7FFF {
		if bSig<<1 != 0 {
			return propagateFloatX80NaN(a, b)
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
			return propagateFloatX80NaN(a, b)
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
	return normalizeRoundAndPackFloatX80(
		RoundingPrecision, zSign, zExp, zSig0, zSig1)

}

// Returns the result of multiplying the extended double-precision floating-
// point values `a' and `b'.  The operation is performed according to the
// IEC/IEEE Standard for Binary Floating-Point Arithmetic.
func (a X80) Mul(b X80) X80 {
	aSig, aExp, aSign := a.frac(), a.exp(), a.sign()
	bSig, bExp, bSign := b.frac(), b.exp(), b.sign()
	zSign := aSign != bSign

	if aExp == 0x7FFF {
		if aSig<<1 != 0 || (bExp == 0x7FFF && bSig<<1 != 0) {
			return propagateFloatX80NaN(a, b)
		}
		if bExp == 0 && bSig == 0 {
			Raise(ExceptionInvalid)
			return X80NaN
		}
		return packFloatX80(zSign, 0x7FFF, 0x8000000000000000)
	}

	if bExp == 0x7FFF {
		if bSig<<1 != 0 {
			return propagateFloatX80NaN(a, b)
		}
		if aExp == 0 && aSig == 0 {
			Raise(ExceptionInvalid)
			return X80NaN
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
	zExp := aExp + bExp - 0x3FFE
	zSig0, zSig1 := mul64To128(aSig, bSig)
	if 0 < zSig0 {
		zSig0, zSig1 = shortShift128Left(zSig0, zSig1, 1)
		zExp--
	}
	return roundAndPackFloatX80(RoundingPrecision, zSign, zExp, zSig0, zSig1)
}

// Returns the result of dividing the extended double-precision floating-point
// value `a' by the corresponding value `b'.  The operation is performed
// according to the IEC/IEEE Standard for Binary Floating-Point Arithmetic.
func (a X80) Div(b X80) X80 {
	aSig, aExp, aSign := a.frac(), a.exp(), a.sign()
	bSig, bExp, bSign := b.frac(), b.exp(), b.sign()
	zSign := aSign != bSign
	if aExp == 0x7FFF {
		if uint64(aSig<<1) != 0 {
			return propagateFloatX80NaN(a, b)
		}
		if bExp == 0x7FFF {
			if uint64(bSig<<1) != 0 {
				return propagateFloatX80NaN(a, b)
			}
			Raise(ExceptionInvalid)
			return X80NaN
		}
		return packFloatX80(zSign, 0x7FFF, 0x8000000000000000)
	}
	if bExp == 0x7FFF {
		if bSig<<1 != 0 {
			return propagateFloatX80NaN(a, b)
		}
		return packFloatX80(zSign, 0, 0)
	}
	if bExp == 0 {
		if bSig == 0 {
			if aExp != 0 && aSig != 0 {
				Raise(ExceptionInvalid)
				return X80NaN
			}
			Raise(ExceptionDivbyzero)
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
		if rem1 != 0 && rem2 != 0 {
			zSig1 |= 1
		}
	}
	return roundAndPackFloatX80(RoundingPrecision, zSign, zExp, zSig0, zSig1)
}

// Returns the remainder of the extended double-precision floating-point value
// `a' with respect to the corresponding value `b'.  The operation is performed
// according to the IEC/IEEE Standard for Binary Floating-Point Arithmetic.
func (a X80) Rem(b X80) X80 {
	aSig0, aExp, aSign := a.frac(), a.exp(), a.sign()
	bSig, bExp := b.frac(), b.exp()
	var term0, term1, q uint64

	if aExp == 0x7FFF {
		if aSig0<<1 != 0 || (bExp == 0x7FFF && bSig<<1 != 0) {
			return propagateFloatX80NaN(a, b)
		}
		Raise(ExceptionInvalid)
		return X80NaN
	}
	if bExp == 0x7FFF {
		if bSig<<1 != 0 {
			return propagateFloatX80NaN(a, b)
		}
		return a
	}
	if bExp == 0 {
		if bSig == 0 {
			Raise(ExceptionInvalid)
			return X80NaN
		}
		bExp, bSig = normalizeFloatX80Subnormal(bSig)
	}
	if aExp == 0 {
		if aSig0<<1 == 0 {
			return a
		}
		aExp, aSig0 = normalizeFloatX80Subnormal(aSig0)
	}
	bSig |= 0x8000000000000000
	zSign := aSign
	expDiff := aExp - bExp
	aSig1 := uint64(0)
	if expDiff < 0 {
		if expDiff < -1 {
			return a
		}
		aSig0, aSig1 = shift128Right(aSig0, 0, 1)
		expDiff = 0
	}
	if bSig <= aSig0 {
		aSig0 -= bSig
	}
	expDiff -= 64
	for 0 < expDiff {
		q := estimateDiv128To64(aSig0, aSig1, bSig)
		if 2 < q {
			q -= 2
		} else {
			q = 0
		}
		term0, term1 = mul64To128(bSig, q)
		aSig0, aSig1 = sub128(aSig0, aSig1, term0, term1)
		aSig0, aSig1 = shortShift128Left(aSig0, aSig1, 62)
		expDiff -= 62
	}
	expDiff += 64
	if 0 < expDiff {
		q := estimateDiv128To64(aSig0, aSig1, bSig)
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
	} else {
		term1 = 0
		term0 = bSig
	}
	alternateASig0, alternateASig1 := sub128(term0, term1, aSig0, aSig1)
	if lt128(alternateASig0, alternateASig1, aSig0, aSig1) ||
		eq128(alternateASig0, alternateASig1, aSig0, aSig1) &&
			(q&1) != 0 {
		aSig0 = alternateASig0
		aSig1 = alternateASig1
		zSign = !zSign
	}
	return normalizeRoundAndPackFloatX80(80, zSign, bExp+expDiff, aSig0, aSig1)
}

// Returns the square root of the extended double-precision floating-point
// value `a'.  The operation is performed according to the IEC/IEEE Standard
// for Binary Floating-Point Arithmetic.
func (a X80) Sqrt() X80 {
	aSig0, aExp, aSign := a.frac(), a.exp(), a.sign()
	var aSig1 uint64
	if aExp == 0x7FFF {
		if aSig0<<1 != 0 {
			return propagateFloatX80NaN(a, a)
		}
		if !aSign {
			return a
		}
		Raise(ExceptionInvalid)
		return X80NaN
	}
	if aSign {
		if aExp != 0 && aSig0 != 0 {
			return a
		}
		Raise(ExceptionInvalid)
		return X80NaN
	}
	if aExp == 0 {
		if aSig0 == 0 {
			return X80Zero
		}
		aExp, aSig0 = normalizeFloatX80Subnormal(aSig0)
	}
	zExp := ((aExp - 0x3FFF) >> 1) + 0x3FFF
	zSig0 := uint64(estimateSqrt32(aExp, uint32(aSig0>>32)))
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
	return roundAndPackFloatX80(RoundingPrecision, false, zExp, zSig0, zSig1)
}

// Software IEC/IEEE extended double-precision operations.
func (a X80) Lognp1() X80 {
	// TODO
	panic("not implemented")
}

// Software IEC/IEEE extended double-precision operations.
func (a X80) Logn() X80 {
	// TODO
	panic("not implemented")
}

// Software IEC/IEEE extended double-precision operations.
func (a X80) Log2() X80 {
	// TODO
	panic("not implemented")
}

// Software IEC/IEEE extended double-precision operations.
func (a X80) Log10() X80 {
	// TODO
	panic("not implemented")
}
