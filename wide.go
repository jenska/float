package float

import (
	"math"
	"math/bits"
)

// wide is an internal floating-point format with a 128-bit significand in
// which the transcendental functions are evaluated before their result is
// rounded once to X80.  Its value is (-1)^neg * (hi:lo) * 2^(exp-127): the
// top bit of hi has weight 2^exp and is set for every nonzero value.  The
// exponent is unbounded and the operations truncate, so every operation has a
// relative error below 2^-126; no exceptions are raised.
type wide struct {
	neg    bool
	exp    int
	hi, lo uint64
}

var (
	wOne = wideFromInt(1)
	wTwo = wideFromInt(2)
)

// normWide returns the normalized wide value (-1)^neg * (hi:lo) * 2^(exp-127).
func normWide(neg bool, exp int, hi, lo uint64) wide {
	if hi == 0 {
		if lo == 0 {
			return wide{}
		}
		hi, lo, exp = lo, 0, exp-64
	}
	if s := bits.LeadingZeros64(hi); s != 0 {
		hi, lo, exp = hi<<s|lo>>(64-s), lo<<s, exp-s
	}
	return wide{neg, exp, hi, lo}
}

// wideFromX80 converts the finite canonical a exactly.
func wideFromX80(a X80) wide {
	exp := a.exp()
	if exp == 0 {
		exp = 1
	}
	return normWide(a.sign(), exp-0x3FFF, a.frac(), 0)
}

func wideFromInt(k int) wide {
	if k < 0 {
		return normWide(true, 63, uint64(-k), 0)
	}
	return normWide(false, 63, uint64(k), 0)
}

// wideConst returns the 128-bit truncation of a Constant.
func wideConst(c Constant) wide {
	k := constantBits[c]
	return wide{exp: k.exp - 0x3FFF, hi: k.hi, lo: k.lo}
}

// float64 returns a approximately, for values within the float64 range.
func (a wide) float64() float64 {
	f := math.Ldexp(float64(a.hi), a.exp-63)
	if a.neg {
		return -f
	}
	return f
}

// wideFromFloat64 converts the finite nonzero f, scaled by 2^scale.
func wideFromFloat64(f float64, scale int) wide {
	frac, exp := math.Frexp(f) // f = frac * 2^exp, |frac| in [0.5, 1)
	m := uint64(math.Abs(frac) * (1 << 53))
	return normWide(f < 0, exp+scale+10, m, 0)
}

func (a wide) isZero() bool { return a.hi == 0 }

func (a wide) negate() wide {
	a.neg = !a.neg
	return a
}

func (a wide) abs() wide {
	a.neg = false
	return a
}

// scale returns a * 2^n.
func (a wide) scale(n int) wide {
	if a.hi != 0 {
		a.exp += n
	}
	return a
}

// cmpAbs compares |a| and |b|.
func (a wide) cmpAbs(b wide) int {
	switch {
	case a.hi == 0 && b.hi == 0:
		return 0
	case a.hi == 0:
		return -1
	case b.hi == 0:
		return 1
	case a.exp != b.exp:
		return cmpInt(a.exp, b.exp)
	case a.hi != b.hi:
		return cmpUint(a.hi, b.hi)
	}
	return cmpUint(a.lo, b.lo)
}

func cmpInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func cmpUint(a, b uint64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// shr128 shifts hi:lo right by n, discarding the bits shifted out.
func shr128(hi, lo uint64, n int) (uint64, uint64) {
	switch {
	case n == 0:
		return hi, lo
	case n < 64:
		return hi >> n, lo>>n | hi<<(64-n)
	case n < 128:
		return 0, hi >> (n - 64)
	}
	return 0, 0
}

func (a wide) add(b wide) wide {
	if b.hi == 0 {
		return a
	}
	if a.hi == 0 {
		return b
	}
	if a.cmpAbs(b) < 0 {
		a, b = b, a
	}
	bh, bl := shr128(b.hi, b.lo, a.exp-b.exp)
	if a.neg == b.neg {
		lo, c := bits.Add64(a.lo, bl, 0)
		hi, c := bits.Add64(a.hi, bh, c)
		if c != 0 {
			return wide{a.neg, a.exp + 1, hi>>1 | 1<<63, lo>>1 | hi<<63}
		}
		return wide{a.neg, a.exp, hi, lo}
	}
	lo, borrow := bits.Sub64(a.lo, bl, 0)
	hi, _ := bits.Sub64(a.hi, bh, borrow)
	return normWide(a.neg, a.exp, hi, lo)
}

func (a wide) sub(b wide) wide {
	return a.add(b.negate())
}

func (a wide) mul(b wide) wide {
	if a.hi == 0 || b.hi == 0 {
		return wide{}
	}
	// the top 128 bits of the 256-bit product, ignoring lo*lo
	hi, lo := bits.Mul64(a.hi, b.hi)
	h2, _ := bits.Mul64(a.hi, b.lo)
	h3, _ := bits.Mul64(a.lo, b.hi)
	var c uint64
	lo, c = bits.Add64(lo, h2, 0)
	hi += c
	lo, c = bits.Add64(lo, h3, 0)
	hi += c
	exp := a.exp + b.exp
	if hi>>63 == 0 {
		hi, lo = hi<<1|lo>>63, lo<<1
	} else {
		exp++
	}
	return wide{a.neg != b.neg, exp, hi, lo}
}

// divInt returns a / n for n > 0.
func (a wide) divInt(n uint64) wide {
	q1, r := bits.Div64(0, a.hi, n)
	q0, _ := bits.Div64(r, a.lo, n)
	return normWide(a.neg, a.exp, q1, q0)
}

// recip returns 1/a for nonzero a by Newton's iteration from a float64
// estimate; each step doubles the number of correct bits.
func (a wide) recip() wide {
	m := a
	m.exp = 0 // |m| in [1, 2)
	y := wideFromFloat64(1/m.float64(), -a.exp)
	for range 2 {
		e := wOne.sub(a.mul(y))
		y = y.add(y.mul(e))
	}
	return y
}

func (a wide) div(b wide) wide {
	return a.mul(b.recip())
}

// sqrt returns the square root of a >= 0.
func (a wide) sqrt() wide {
	if a.hi == 0 {
		return a
	}
	// a = m * 2^(2h) with m in [1, 4)
	h := a.exp >> 1
	m := a
	m.exp -= 2 * h
	// y ~ 1/sqrt(a), refined with y += y*(1 - a*y^2)/2
	y := wideFromFloat64(1/math.Sqrt(m.float64()), -h)
	for range 2 {
		e := wOne.sub(a.mul(y).mul(y))
		y = y.add(y.mul(e).scale(-1))
	}
	// s = a*y, corrected with s += y*(a - s^2)/2
	s := a.mul(y)
	return s.add(y.mul(a.sub(s.mul(s))).scale(-1))
}

// round returns a, an approximation of an irrational value, rounded to X80
// with the current rounding mode and precision.  It raises the inexact
// exception and, if the result is out of range, overflow or underflow.
func (a wide) round(e *Env) X80 {
	if a.hi == 0 {
		return packFloatX80(a.neg, 0, 0)
	}
	zExp := max(min(a.exp+0x3FFF, 0x10000), -0x7000)
	return e.roundAndPackFloatX80(e.RoundingPrecision, a.neg, zExp, a.hi, a.lo|1)
}

// horner returns c[0] + x*(c[1] + x*(c[2] + ...)).
func horner(x wide, c []wide) wide {
	p := c[len(c)-1]
	for i := len(c) - 2; i >= 0; i-- {
		p = c[i].add(x.mul(p))
	}
	return p
}
