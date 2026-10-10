package float

// Eq returns true if the extended double-precision floating-point value `a' is
// equal to the corresponding value `b', and false otherwise.  The comparison is
// performed according to the IEC/IEEE Standard for Binary Floating-Point
// Arithmetic.
func (e *Env) Eq(a X80, b X80) bool {
	a, b = a.canonical(), b.canonical()
	if (a.exp() == 0x7FFF && a.frac()<<1 != 0) || (b.exp() == 0x7FFF && b.frac()<<1 != 0) {
		if a.IsSignalingNaN() || b.IsSignalingNaN() {
			e.Raise(ExceptionInvalid)
		}
		return false
	}
	return a.low == b.low && (a.high == b.high || (a.low == 0 && (a.high|b.high)<<1 == 0))
}

// Gt returns true if the extended double-precision floating-point value `a' is greater
// than the corresponding value `b', and false otherwise.  The invalid exception
// is raised if either operand is a NaN.
func (e *Env) Gt(a X80, b X80) bool {
	return e.Lt(b, a)
}

// Le returns true if the extended double-precision floating-point value `a' is less than or
// equal to the corresponding value `b', and false otherwise.
func (e *Env) Le(a X80, b X80) bool {
	a, b = a.canonical(), b.canonical()
	if (a.exp() == 0x7FFF && a.frac()<<1 != 0) || (b.exp() == 0x7FFF && b.frac()<<1 != 0) {
		e.Raise(ExceptionInvalid)
		return false
	}
	aSign, bSign := a.sign(), b.sign()
	if aSign != bSign {
		return aSign || ((a.high|b.high)<<1 == 0 && (a.low|b.low) == 0)
	}
	if aSign {
		return le128(uint64(b.high), b.low, uint64(a.high), a.low)
	}
	return le128(uint64(a.high), a.low, uint64(b.high), b.low)
}

// Ge returns true if the extended double-precision floating-point value `a' is greater than or
// equal to the corresponding value `b', and false otherwise.  The invalid
// exception is raised if either operand is a NaN.
func (e *Env) Ge(a X80, b X80) bool {
	return e.Le(b, a)
}

// Lt returns true if the extended double-precision floating-point value `a' is
// less than the corresponding value `b', and false otherwise.  The comparison
// is performed according to the IEC/IEEE Standard for Binary Floating-Point
// Arithmetic.
func (e *Env) Lt(a X80, b X80) bool {
	a, b = a.canonical(), b.canonical()
	if (a.exp() == 0x7FFF && a.frac()<<1 != 0) || (b.exp() == 0x7FFF && b.frac()<<1 != 0) {
		e.Raise(ExceptionInvalid)
		return false
	}
	aSign, bSign := a.sign(), b.sign()
	if aSign != bSign {
		return aSign && ((a.high|b.high)<<1 != 0 || (a.low|b.low) != 0)
	}
	if aSign {
		return lt128(uint64(b.high), b.low, uint64(a.high), a.low)
	}
	return lt128(uint64(a.high), a.low, uint64(b.high), b.low)
}

// EqSignaling returns true if the extended double-precision floating-point value `a' is equal
// to the corresponding value `b', and false otherwise.  The invalid exception is
// raised if either operand is a NaN.  Otherwise, the comparison is performed
// according to the IEC/IEEE Standard for Binary Floating-Point Arithmetic.
func (e *Env) EqSignaling(a X80, b X80) bool {
	a, b = a.canonical(), b.canonical()
	if (a.exp() == 0x7FFF && a.frac()<<1 != 0) || (b.exp() == 0x7FFF && b.frac()<<1 != 0) {
		e.Raise(ExceptionInvalid)
		return false
	}
	return a.low == b.low && (a.high == b.high || (a.low == 0 && (a.high|b.high)<<1 == 0))
}

// GtQuiet returns true if the extended double-precision floating-point value `a' is
// greater than the corresponding value `b', and false otherwise.  Quiet NaNs
// do not cause an exception.  Otherwise, the comparison is performed according
// to the IEC/IEEE Standard for Binary Floating-Point Arithmetic.
func (e *Env) GtQuiet(a X80, b X80) bool {
	return e.LtQuiet(b, a)
}

// LeQuiet returns true if the extended double-precision floating-point value `a' is less
// than or equal to the corresponding value `b', and false otherwise.  Quiet NaNs
// do not cause an exception.  Otherwise, the comparison is performed according
// to the IEC/IEEE Standard for Binary Floating-Point Arithmetic.
func (e *Env) LeQuiet(a X80, b X80) bool {
	a, b = a.canonical(), b.canonical()
	if (a.exp() == 0x7FFF && a.frac()<<1 != 0) || (b.exp() == 0x7FFF && b.frac()<<1 != 0) {
		if a.IsSignalingNaN() || b.IsSignalingNaN() {
			e.Raise(ExceptionInvalid)
		}
		return false
	}
	aSign, bSign := a.sign(), b.sign()
	if aSign != bSign {
		return aSign || ((a.high|b.high)<<1 == 0 && (a.low|b.low) == 0)
	}
	if aSign {
		return le128(uint64(b.high), b.low, uint64(a.high), a.low)
	}
	return le128(uint64(a.high), a.low, uint64(b.high), b.low)
}

// GeQuiet returns true if the extended double-precision floating-point value `a' is greater
// than or equal to the corresponding value `b', and false otherwise.  Quiet NaNs
// do not cause an exception.  Otherwise, the comparison is performed according
// to the IEC/IEEE Standard for Binary Floating-Point Arithmetic.
func (e *Env) GeQuiet(a X80, b X80) bool {
	return e.LeQuiet(b, a)
}

// LtQuiet returns true if the extended double-precision floating-point value `a' is less
// than the corresponding value `b', and false otherwise.  Quiet NaNs do not cause
// an exception.  Otherwise, the comparison is performed according to the
// IEC/IEEE Standard for Binary Floating-Point Arithmetic.
func (e *Env) LtQuiet(a X80, b X80) bool {
	a, b = a.canonical(), b.canonical()
	if (a.exp() == 0x7FFF && a.frac()<<1 != 0) || (b.exp() == 0x7FFF && b.frac()<<1 != 0) {
		if a.IsSignalingNaN() || b.IsSignalingNaN() {
			e.Raise(ExceptionInvalid)
		}
		return false
	}
	aSign, bSign := a.sign(), b.sign()
	if aSign != bSign {
		return aSign && (((a.high|b.high)<<1 != 0) || (a.low|b.low) != 0)
	}
	if aSign {
		return lt128(uint64(b.high), b.low, uint64(a.high), a.low)
	}
	return lt128(uint64(a.high), a.low, uint64(b.high), b.low)
}
