package float

import (
	"math/big"
	"math/bits"
	"strconv"
	"strings"
	"sync"
)

// More "constants" for X80 format, correctly rounded to 64 significand bits.
var (
	X80Ln10     = newFromHexString("4000935D8DDDAAA8AC17") // Ln(10)
	X80Log10E   = newFromHexString("3FFDDE5BD8A937287195") // Log10(e)
	X80Log10Of2 = newFromHexString("3FFD9A209A84FBCFF799") // Log10(2)
)

// pow10Exact holds 10^n for 0 <= n <= 27, the powers of ten that are exact
// X80 values (5^27 < 2^63).
var pow10Exact = func() (t [28]X80) {
	five := int64(1)
	for n := range t {
		t[n] = Int64ToFloatX80(five)
		t[n].high += uint16(n) // * 2^n
		five *= 5
	}
	return t
}()

// Constant identifies a mathematical constant that Value rounds under the
// current rounding mode and precision.
type Constant int

// The constants Value can produce.
const (
	ConstPi       Constant = iota // pi
	ConstE                        // e
	ConstLn2                      // ln(2)
	ConstLn10                     // ln(10)
	ConstLog2E                    // log2(e)
	ConstLog10E                   // log10(e)
	ConstLog10Of2                 // log10(2)
)

// constantBits holds the biased exponent and the first 128 significand bits
// of each Constant. All of them are irrational, so the bits beyond are never
// all zero and act as a sticky bit.
var constantBits = [...]struct {
	exp    int
	hi, lo uint64
}{
	ConstPi:       {0x4000, 0xC90FDAA22168C234, 0xC4C6628B80DC1CD1},
	ConstE:        {0x4000, 0xADF85458A2BB4A9A, 0xAFDC5620273D3CF1},
	ConstLn2:      {0x3FFE, 0xB17217F7D1CF79AB, 0xC9E3B39803F2F6AF},
	ConstLn10:     {0x4000, 0x935D8DDDAAA8AC16, 0xEA56D62B82D30A28},
	ConstLog2E:    {0x3FFF, 0xB8AA3B295C17F0BB, 0xBE87FED0691D3E88},
	ConstLog10E:   {0x3FFD, 0xDE5BD8A937287195, 0x355BAAAFAD33DC32},
	ConstLog10Of2: {0x3FFD, 0x9A209A84FBCFF798, 0x8F8959AC0B7C9178},
}

// Value returns c correctly rounded to the current RoundingPrecision using the
// current RoundingMode, and raises the inexact exception. The X80Pi, X80E, ...
// variables hold the round-to-nearest values at full precision.
func (e *Env) Constant(c Constant) X80 {
	k := constantBits[c]
	return e.roundAndPackFloatX80(e.RoundingPrecision, false, k.exp, k.hi, k.lo|1)
}

// Pow10 returns 10^n correctly rounded to the current RoundingPrecision using
// the current RoundingMode. Results that cannot be represented exactly raise
// the inexact exception, and out-of-range results overflow or underflow like
// any other operation. 10^n is exact for 0 <= n <= 27.
func (e *Env) Pow10(n int) X80 {
	switch {
	case 0 <= n && n < len(pow10Exact):
		if e.RoundingPrecision != 80 {
			return e.RoundToPrecision(pow10Exact[n], e.RoundingPrecision)
		}
		return pow10Exact[n]
	case -len(pow10Exact) < n && n < 0:
		return e.Div(X80One, pow10Exact[-n]) // one correctly rounded operation
	}
	n = max(min(n, 5000), -5000) // beyond this the result is Inf or 0 anyway
	if z, ok := e.roundWideSafe(false, wPow10(n)); ok {
		return z
	}
	p := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(max(n, -n))), nil)
	if n < 0 {
		return e.roundRatio(false, big.NewInt(1), p)
	}
	return e.roundRatio(false, p, big.NewInt(1))
}

// pow10Tables holds 10^(2^i) and 10^-(2^i) for i = 0..12, truncated to 128
// bits.
var pow10Tables = sync.OnceValues(func() (pos, neg [13]wide) {
	for i := range pos {
		p := new(big.Int).Exp(big.NewInt(10), big.NewInt(1<<i), nil)
		pos[i] = wideFromBig(p, 0)
		l := uint(p.BitLen() + 128)
		q := new(big.Int).Lsh(big.NewInt(1), l)
		neg[i] = wideFromBig(q.Quo(q, p), -int(l))
	}
	return pos, neg
})

// wideFromBig returns x * 2^scale truncated to 128 bits.
func wideFromBig(x *big.Int, scale int) wide {
	sh := max(x.BitLen()-128, 0)
	t := new(big.Int).Rsh(x, uint(sh))
	return normWide(false, 127+sh+scale, new(big.Int).Rsh(t, 64).Uint64(), t.Uint64())
}

// wPow10 returns 10^n for |n| < 8192 with a relative error below 2^-120.
func wPow10(n int) wide {
	pos, neg := pow10Tables()
	t := &pos
	if n < 0 {
		t, n = &neg, -n
	}
	z := wOne
	for i := 0; n != 0; i, n = i+1, n>>1 {
		if n&1 != 0 {
			z = z.mul(t[i])
		}
	}
	return z
}

// wideSlack bounds the error of a wide product of decimal conversions, in
// units of the last bit of its significand.
const wideSlack = 1 << 20

// roundWideSafe rounds the approximation p of an exact value to X80 with the
// current rounding mode and precision, if p is close enough to the exact
// value that the rounding cannot differ: its discarded bits must be farther
// than wideSlack from zero, from a half-way point and from the next
// representable value. Results near the subnormal or overflow range are left
// to the exact path.
func (e *Env) roundWideSafe(neg bool, p wide) (X80, bool) {
	zExp := p.exp + 0x3FFF
	if p.hi == 0 || zExp <= 64 || zExp >= 0x7FFE {
		return X80{}, false
	}
	s := uint(0) // significand bits discarded from hi
	switch e.RoundingPrecision {
	case 64:
		s = 11
	case 32:
		s = 40
	}
	// the discarded part as a 128-bit value dHi:lo, and the half-way point
	dHi := p.hi & (1<<s - 1)
	halfHi, halfLo := uint64(0), uint64(1)<<63
	if s > 0 {
		halfHi, halfLo = 1<<(s-1), 0
	}
	near := func(h, l uint64) bool { // |dHi:lo - h:l| < wideSlack
		dl, b := bits.Sub64(p.lo, l, 0)
		dh, _ := bits.Sub64(dHi, h, b)
		if dh == 0 {
			return dl < wideSlack
		}
		return dh == ^uint64(0) && dl > ^uint64(0)-wideSlack
	}
	if near(0, 0) || near(1<<s, 0) || near(halfHi, halfLo) {
		return X80{}, false
	}
	return e.roundAndPackFloatX80(e.RoundingPrecision, neg, zExp, p.hi, p.lo), true
}

// roundRatio returns the positive rational num/den, negated if zSign is set,
// correctly rounded to the current RoundingPrecision using the current
// RoundingMode.
func (e *Env) roundRatio(zSign bool, num, den *big.Int) X80 {
	// scale so that the integer quotient has 128 or 129 bits
	s := 128 - (num.BitLen() - den.BitLen())
	n, d := new(big.Int).Set(num), new(big.Int).Set(den)
	if s > 0 {
		n.Lsh(n, uint(s))
	} else {
		d.Lsh(d, uint(-s))
	}
	q, r := n.QuoRem(n, d, new(big.Int))
	sticky := r.Sign() != 0
	if q.BitLen() > 128 {
		sticky = sticky || q.Bit(0) != 0
		q.Rsh(q, 1)
		s--
	}
	hi := new(big.Int).Rsh(q, 64).Uint64()
	lo := q.Uint64() // low 64 bits
	if sticky {
		lo |= 1
	}
	zExp := 127 - s + 0x3FFF
	zExp = max(min(zExp, 0x10000), -0x7000)
	return e.roundAndPackFloatX80(e.RoundingPrecision, zSign, zExp, hi, lo)
}

// Parse converts the string s to the nearest X80 value, rounded according to
// the current rounding mode and precision; the inexact exception is raised if
// the conversion is not exact. Overflow and underflow raise the corresponding
// exceptions instead of returning an error.
//
// s may have a sign and is either a decimal number with an optional decimal
// point and exponent ("-12.5e-3"), a hexadecimal number with an optional
// binary exponent ("0x1.8p3"), or "Inf", "Infinity" or "NaN" in any case.
// Parse returns a *strconv.NumError with ErrSyntax if s is malformed.
func (e *Env) Parse(s string) (X80, error) {
	str := s
	neg := false
	if str != "" && (str[0] == '+' || str[0] == '-') {
		neg = str[0] == '-'
		str = str[1:]
	}
	switch strings.ToLower(str) {
	case "inf", "infinity":
		return packFloatX80(neg, 0x7FFF, 1<<63), nil
	case "nan":
		return packFloatX80(neg, 0x7FFF, 0xC000000000000000), nil
	}

	base, expChar := 10, byte('e')
	if len(str) > 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X') {
		base, expChar = 16, 'p'
		str = str[2:]
	}

	// significand digits, with the number of digits after the point; the
	// significand is kept in mHi:mLo until it exceeds 128 bits
	var mHi, mLo uint64
	var mant *big.Int
	digits, fracDigits, sawPoint, i := 0, 0, false, 0
	for ; i < len(str); i++ {
		c := str[i]
		var d int
		switch {
		case c == '.' && !sawPoint:
			sawPoint = true
			continue
		case '0' <= c && c <= '9':
			d = int(c - '0')
		case base == 16 && 'a' <= c|0x20 && c|0x20 <= 'f':
			d = int(c|0x20-'a') + 10
		default:
			goto exponent
		}
		if mant == nil {
			h1, l := bits.Mul64(mLo, uint64(base))
			h2, l2 := bits.Mul64(mHi, uint64(base))
			hi, c1 := bits.Add64(h1, l2, 0)
			lo, c2 := bits.Add64(l, uint64(d), 0)
			hi, c3 := bits.Add64(hi, 0, c2)
			if h2|c1|c3 == 0 {
				mHi, mLo = hi, lo
				goto next
			}
			mant = uint128ToBig(mHi, mLo)
		}
		mant.Mul(mant, big.NewInt(int64(base))).Add(mant, big.NewInt(int64(d)))
	next:
		digits++
		if sawPoint {
			fracDigits++
		}
	}
exponent:
	if digits == 0 {
		return X80{}, &strconv.NumError{Func: "Parse", Num: s, Err: strconv.ErrSyntax}
	}
	exp := 0
	if i < len(str) {
		if str[i]|0x20 != expChar || i+1 == len(str) {
			return X80{}, &strconv.NumError{Func: "Parse", Num: s, Err: strconv.ErrSyntax}
		}
		i++
		expNeg := false
		if str[i] == '+' || str[i] == '-' {
			expNeg = str[i] == '-'
			i++
		}
		if i == len(str) {
			return X80{}, &strconv.NumError{Func: "Parse", Num: s, Err: strconv.ErrSyntax}
		}
		for ; i < len(str); i++ {
			if str[i] < '0' || str[i] > '9' {
				return X80{}, &strconv.NumError{Func: "Parse", Num: s, Err: strconv.ErrSyntax}
			}
			exp = min(exp*10+int(str[i]-'0'), 1_000_000) // saturate
		}
		if expNeg {
			exp = -exp
		}
	}
	if mant == nil && mHi|mLo == 0 {
		return packFloatX80(neg, 0, 0), nil
	}

	one := big.NewInt(1)
	if base == 16 && mant == nil {
		// mant * 2^e is exact: round it directly
		w := normWide(neg, 127, mHi, mLo)
		zExp := max(min(w.exp+exp-4*fracDigits+0x3FFF, 0x10000), -0x7000)
		return e.roundAndPackFloatX80(e.RoundingPrecision, neg, zExp, w.hi, w.lo), nil
	}
	if mant == nil {
		if pe := exp - fracDigits; -8192 < pe && pe < 8192 {
			// fast paths: one exact operation, or a wide product
			if mHi == 0 && mLo < 1<<63 && -len(pow10Exact) < pe && pe < len(pow10Exact) {
				m := Int64ToFloatX80(int64(mLo))
				if neg {
					m = m.Neg()
				}
				if pe >= 0 {
					return e.Mul(m, pow10Exact[pe]), nil
				}
				return e.Div(m, pow10Exact[-pe]), nil
			}
			if z, ok := e.roundWideSafe(neg, normWide(false, 127, mHi, mLo).mul(wPow10(pe))); ok {
				return z, nil
			}
		}
		mant = uint128ToBig(mHi, mLo)
	}
	if base == 16 {
		// mant * 16^-fracDigits * 2^exp; clamp far outside the X80 range
		pe := max(min(exp-4*fracDigits, 40000), -40000)
		if pe >= 0 {
			return e.roundRatio(neg, mant.Lsh(mant, uint(pe)), one), nil
		}
		return e.roundRatio(neg, mant, new(big.Int).Lsh(one, uint(-pe))), nil
	}
	pe := exp - fracDigits
	// mant * 10^pe; values beyond 10^±5000 overflow or underflow anyway
	switch magnitude := pe + digits; {
	case magnitude > 5000:
		pe, mant = 5000, big.NewInt(1)
	case magnitude < -5000:
		pe, mant = -5000, big.NewInt(1)
	}
	p := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(max(pe, -pe))), nil)
	if pe >= 0 {
		return e.roundRatio(neg, mant.Mul(mant, p), one), nil
	}
	return e.roundRatio(neg, mant, p), nil
}

func uint128ToBig(hi, lo uint64) *big.Int {
	z := new(big.Int).SetUint64(hi)
	return z.Lsh(z, 64).Or(z, new(big.Int).SetUint64(lo))
}
