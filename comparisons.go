package float

// Returns true if the extended double-precision floating-point value `a' is
// equal to the corresponding value `b', and false otherwise.  The comparison is
// performed according to the IEC/IEEE Standard for Binary Floating-Point
// Arithmetic.
func (a X80) Eq(b X80) bool {
	if (a.exp() == 0x7FFF && a.frac()<<1 != 0) || (b.exp() == 0x7FFF && b.frac()<<1 != 0) {
		if a.IsSignalingNaN() || b.IsSignalingNaN() {
			Raise(ExceptionInvalid)
		}
		return false
	}
	return (a.low == b.low && a.high == b.high) ||
		(a.low == 0 && (a.high|b.high)<<1 == 0)
}

// Returns true if the extended double-precision floating-point value `a' is less than or
// equal to the corresponding value `b', and false otherwise.
func (a X80) Le(b X80) bool {
	if (a.exp() == 0x7FFF && a.frac()<<1 != 0) || (b.exp() == 0x7FFF && b.frac()<<1 != 0) {
		Raise(ExceptionInvalid)
		return false
	}
	aSign := a.sign()
	bSign := b.sign()
	if aSign != bSign {
		return aSign || ((a.high|b.high)<<1 != 0 || (a.low|b.low) == 0)
	}
	if aSign {
		return le128(uint64(b.high), b.low, uint64(a.high), a.low)
	}
	return le128(uint64(a.high), a.low, uint64(b.high), b.low)
}

// Returns true if the extended double-precision floating-point value `a' is
// less than the corresponding value `b', and false otherwise.  The comparison
// is performed according to the IEC/IEEE Standard for Binary Floating-Point
// Arithmetic.
func (a X80) Lt(b X80) bool {
	if (a.exp() == 0x7FFF && a.frac()<<1 != 0) || (b.exp() == 0x7FFF && b.frac()<<1 != 0) {
		Raise(ExceptionInvalid)
		return false
	}
	aSign := a.sign()
	bSign := b.sign()
	if aSign != bSign {
		return aSign && ((a.high|b.high)<<1 != 0 || (a.low|b.low) != 0)
	}
	if aSign {
		return lt128(uint64(b.high), b.low, uint64(a.high), a.low)
	}
	return lt128(uint64(a.high), a.low, uint64(b.high), b.low)
}

// Returns true if the extended double-precision floating-point value `a' is equal
// to the corresponding value `b', and false otherwise.  The invalid exception is
// raised if either operand is a NaN.  Otherwise, the comparison is performed
// according to the IEC/IEEE Standard for Binary Floating-Point Arithmetic.
func (a X80) EqSignaling(b X80) bool {
	if (a.exp() == 0x7FFF && a.frac()<<1 != 0) || (b.exp() == 0x7FFF && b.frac()<<1 != 0) {
		Raise(ExceptionInvalid)
		return false
	}
	return a.low == b.low && (a.high == b.high || (a.low == 0 && (a.high|b.high)<<1 == 0))
}

// Returns true if the extended double-precision floating-point value `a' is less
// than or equal to the corresponding value `b', and false otherwise.  Quiet NaNs
// do not cause an exception.  Otherwise, the comparison is performed according
// to the IEC/IEEE Standard for Binary Floating-Point Arithmetic.
func (a X80) LeQuiet(b X80) bool {
	if (a.exp() == 0x7FFF && a.frac()<<1 != 0) || (b.exp() == 0x7FFF && b.frac()<<1 != 0) {
		if a.IsSignalingNaN() || b.IsSignalingNaN() {
			Raise(ExceptionInvalid)
		}
		return false
	}
	aSign := a.sign()
	bSign := b.sign()
	if aSign != bSign {
		return aSign || (((a.high|b.high)<<1 != 0) || (a.low|b.low) == 0)
	}
	if aSign {
		return le128(uint64(b.high), b.low, uint64(a.high), a.low)
	}
	return le128(uint64(a.high), a.low, uint64(b.high), b.low)
}

// Returns true if the extended double-precision floating-point value `a' is less
// than the corresponding value `b', and false otherwise.  Quiet NaNs do not cause
// an exception.  Otherwise, the comparison is performed according to the
// IEC/IEEE Standard for Binary Floating-Point Arithmetic.
func (a X80) LtQuiet(b X80) bool {
	if (a.exp() == 0x7FFF && a.frac()<<1 != 0) || (b.exp() == 0x7FFF && b.frac()<<1 != 0) {
		if a.IsSignalingNaN() || b.IsSignalingNaN() {
			Raise(ExceptionInvalid)
		}
		return false
	}
	aSign := a.sign()
	bSign := b.sign()
	if aSign != bSign {
		return aSign && (((a.high|b.high)<<1 != 0) || (a.low|b.low) != 0)
	}
	if aSign {
		return lt128(uint64(b.high), b.low, uint64(a.high), a.low)
	}
	return lt128(uint64(a.high), a.low, uint64(b.high), b.low)
}
