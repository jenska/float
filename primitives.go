package float

// Shifts `a' right by the number of bits given in `count'.  If any nonzero
// bits are shifted off, they are ``jammed'' into the least significant bit of
// the result by setting the least significant bit to 1.  The value of `count'
// can be arbitrarily large; in particular, if `count' is greater than 64, the
// result will be either 0 or 1, depending on whether `a' is zero or nonzero.
// The result is stored in the location pointed to by `zPtr'.
func shift64RightJamming(a uint64, count int16) (z uint64) {
	if count == 0 {
		z = a
	} else if count < 64 {
		z = a >> count
		if (a << ((-count) & 63)) != 0 {
			z |= 1
		}
	} else if a != 0 {
		z = 1
	}
	return
}

// Shifts the 128-bit value formed by concatenating `a0' and `a1' right by 64
// _plus_ the number of bits given in `count'.  The shifted result is at most
// 64 nonzero bits; this is stored at the location pointed to by `z0Ptr'.  The
// bits shifted off form a second 64-bit result as follows:  The _last_ bit
// shifted off is the most-significant bit of the extra result, and the other
// 63 bits of the extra result are all zero if and only if _all_but_the_last_
// bits shifted off were all zero.  This extra result is stored in the location
// pointed to by `z1Ptr'.  The value of `count' can be arbitrarily large.
//     (This routine makes more sense if `a0' and `a1' are considered to form
// a fixed-point value with binary point between `a0' and `a1'.  This fixed-
// point value is shifted right by the number of bits given in `count', and
// the integer part of the result is returned.
func shift64ExtraRightJamming(a0, a1 uint64, count int16) (z0, z1 uint64) {
	if count == 0 {
		z1 = a1
		z0 = a0
	} else if count < 64 {
		z1 = (a0 << ((-count) & 63))
		if a1 != 0 {
			z1 |= 1
		}
		z0 = a0 >> count
	} else {
		if count == 64 {
			z1 = a0
			if a1 != 0 {
				z1 |= 1
			}
		} else if (a0 | a1) != 0 {
			z1 = 1
		}
		z0 = 0
	}
	return
}

// Shifts the 128-bit value formed by concatenating `a0' and `a1' right by the
// number of bits given in `count'.  Any bits shifted off are lost.  The value
// of `count' can be arbitrarily large; in particular, if `count' is greater
// than 128, the result will be 0.
func shift128Right(a0, a1 uint64, count int16) (z0, z1 uint64) {
	negCount := (-count) & 63

	if count == 0 {
		z1 = a1
		z0 = a0
	} else if count < 64 {
		z1 = (a0 << negCount) | (a1 >> count)
		z0 = a0 >> count
	} else {
		z1 = 0
		if count < 64 {
			z1 = a0 >> (count & 63)
		}
		z0 = 0
	}
	return
}

// Shifts the 128-bit value formed by concatenating `a0' and `a1' right by the
// number of bits given in `count'.  If any nonzero bits are shifted off, they
// are ``jammed'' into the least significant bit of the result by setting the
// least significant bit to 1.  The value of `count' can be arbitrarily large;
// in particular, if `count' is greater than 128, the result will be either
// 0 or 1, depending on whether the concatenation of `a0' and `a1' is zero or
// nonzero.
func shift128RightJamming(a0, a1 uint64, count int16) (z0, z1 uint64) {
	negCount := (-count) & 63

	if count == 0 {
		z1 = a1
		z0 = a0
	} else if count < 64 {
		z1 = a0<<negCount | a1>>count
		if a1<<uint64(negCount) != 0 {
			z1 |= 1
		}
		z0 = a0 >> count
	} else {
		if count == 64 {
			z1 = a0
			if a1 != 0 {
				z1 |= 1
			}
		} else if count < 128 {
			z1 = a0 >> (count & 63)
			if (a0<<negCount)|a1 != 0 {
				z1 |= 1
			}
		} else if (a0 | a1) != 0 {
			z1 = 1
		}
		z0 = 0
	}
	return
}

// Shifts the 128-bit value formed by concatenating `a0' and `a1' left by the
// number of bits given in `count'.  Any bits shifted off are lost.  The value
// of `count' must be less than 64.  The result is broken into two 64-bit
// pieces
func shortShift128Left(a0, a1 uint64, count int16) (z0, z1 uint64) {
	z1 = a1 << count
	if count == 0 {
		z0 = a0
	} else {
		z0 = (a0 << count) | (a1 >> ((-count) & 63))
	}
	return
}

// Subtracts the 128-bit value formed by concatenating `b0' and `b1' from the
// 128-bit value formed by concatenating `a0' and `a1'.  Subtraction is modulo
// 2^128, so any borrow out (carry out) is lost.
func sub128(a0, a1, b0, b1 uint64) (z0, z1 uint64) {
	z1 = a1 - b1
	z0 = a0 - b0
	if a1 < b1 {
		z0--
	}
	return
}

// Adds the 128-bit value formed by concatenating `a0' and `a1' to the 128-bit
// value formed by concatenating `b0' and `b1'.  Addition is modulo 2^128, so
// any carry out is lost.
func add128(a0, a1, b0, b1 uint64) (z0, z1 uint64) {
	z1 = a1 + b1
	z0 = a0 + b0
	if z1 < a1 {
		z0++
	}
	return
}

func x1(x bool) uint64 {
	if x {
		return 1
	}
	return 0
}

// Adds the 192-bit value formed by concatenating `a0', `a1', and `a2' to the
// 192-bit value formed by concatenating `b0', `b1', and `b2'.  Addition is
// modulo 2^192, so any carry out is lost.
func add192(a0, a1, a2, b0, b1, b2 uint64) (z0, z1, z2 uint64) {
	z2 = a2 + b2
	carry1 := (z2 < a2)
	z1 = a1 + b1
	carry0 := (z1 < a1)
	z0 = a0 + b0
	z1 += x1(carry1)
	z0 += x1(z1 < x1(carry1))
	z0 += x1(carry0)
	return
}

// Subs the 192-bit value formed by concatenating `a0', `a1', and `a2' from the
// 192-bit value formed by concatenating `b0', `b1', and `b2'.  Addition is
// modulo 2^192, so any carry out is lost.
func sub192(a0, a1, a2, b0, b1, b2 uint64) (z0, z1, z2 uint64) {
	z2 = a2 - b2
	borrow1 := a2 < b2
	z1 = a1 - b1
	borrow0 := a1 < b1
	z0 = a0 - b0
	z0 -= x1(z1 < x1(borrow1))
	z1 -= x1(borrow1)
	z0 -= x1(borrow0)
	return
}

// Multiplies `a' by `b' to obtain a 128-bit product.
func mul64To128(a, b uint64) (z0, z1 uint64) {
	aLow := a & 0xffffffff
	aHigh := a >> 32
	bLow := b & 0xffffffff
	bHigh := b >> 32
	z1 = aLow * bLow
	zMiddleA := aLow * bHigh
	zMiddleB := aHigh * bLow
	z0 = aHigh * bHigh
	zMiddleA += zMiddleB
	z0 += zMiddleA>>32 + x1(zMiddleA < zMiddleB)<<32
	zMiddleA <<= 32
	z1 += zMiddleA
	if z1 < zMiddleA {
		z0++
	}
	return
}

// Returns an approximation to the 64-bit integer quotient obtained by dividing
// `b' into the 128-bit value formed by concatenating `a0' and `a1'.  The
// divisor `b' must be at least 2^63.  If q is the exact quotient truncated
// toward zero, the approximation returned lies between q and q + 2 inclusive.
// If the exact quotient q is larger than 64 bits, the maximum positive 64-bit
// unsigned integer is returned.
func estimateDiv128To64(a0, a1, b uint64) uint64 {
	if b <= a0 {
		return 0xFFFFFFFFFFFFFFFF
	}
	b0 := b >> 32
	z := uint64(0xFFFFFFFF00000000)
	if b0<<32 > a0 {
		z = (a0 / b0) << 32
	}
	term0, term1 := mul64To128(b, z)
	rem0, rem1 := sub128(a0, a1, term0, term1)
	for int(rem0) < 0 {
		z -= 0x100000000
		b1 := b << 32
		rem0, rem1 = add128(rem0, rem1, b0, b1)
	}
	rem0 = (rem0 << 32) | (rem1 >> 32)
	if b0<<32 <= rem0 {
		z |= 0xFFFFFFFF
	} else {
		z |= rem0 / b0
	}
	return z
}

// Returns an approximation to the square root of the 32-bit significand given
// by `a'.  Considered as an integer, `a' must be at least 2^31.  If bit 0 of
// `aExp' (the least significant bit) is 1, the integer returned approximates
// 2^31*sqrt(`a'/2^31), where `a' is considered an integer.  If bit 0 of `aExp'
// is 0, the integer returned approximates 2^31*sqrt(`a'/2^30).  In either
// case, the approximation returned lies strictly within +/-2 of the exact
// value.
func estimateSqrt32(aExp int32, a uint32) (z uint32) {
	index := (a >> 27) & 15
	if aExp&1 != 0 {
		z = 0x4000 + (a >> 17) - sqrtOddAdjustments[index]
		z = ((a / z) << 14) + (z << 15)
		a >>= 1
	} else {
		z = 0x8000 + (a >> 17) - sqrtEvenAdjustments[index]
		z = a/z + z
		if 0x20000 <= z {
			z = 0xFFFF8000
		} else {
			z <<= 15
		}
		if z <= a {
			return uint32(int32(a) >> 1)
		}
	}
	return uint32((uint64(a)<<31)/uint64(z)) + (z >> 1)
}

var sqrtOddAdjustments []uint32 = []uint32{
	0x0004, 0x0022, 0x005D, 0x00B1, 0x011D, 0x019F, 0x0236, 0x02E0,
	0x039C, 0x0468, 0x0545, 0x0631, 0x072B, 0x0832, 0x0946, 0x0A67,
}
var sqrtEvenAdjustments []uint32 = []uint32{
	0x0A2D, 0x08AF, 0x075A, 0x0629, 0x051A, 0x0429, 0x0356, 0x029E,
	0x0200, 0x0179, 0x0109, 0x00AF, 0x0068, 0x0034, 0x0012, 0x0002,
}

// Returns true if the 128-bit value formed by concatenating `a0' and `a1'
// is equal to the 128-bit value formed by concatenating `b0' and `b1'.
// Otherwise, returns false.
func eq128(a0, a1, b0, b1 uint64) bool {
	return (a0 == b0) && (a1 == b1)
}

// Returns true if the 128-bit value formed by concatenating `a0' and `a1' is less
// than or equal to the 128-bit value formed by concatenating `b0' and `b1'.
// Otherwise, returns false
func le128(a0, a1, b0, b1 uint64) bool {
	return (a0 < b0) || ((a0 == b0) && (a1 <= b1))
}

// Returns 1 if the 128-bit value formed by concatenating `a0' and `a1' is less
// than the 128-bit value formed by concatenating `b0' and `b1'.  Otherwise,
// returns 0.
func lt128(a0, a1, b0, b1 uint64) bool {
	return (a0 < b0) || ((a0 == b0) && (a1 < b1))
}
