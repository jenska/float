package float

//go:generate go run ./internal/gentables

import (
	"math"
	"math/bits"
)

// Elementary and transcendental functions.
//
// Each function handles special operands exactly, then evaluates its result
// in the 128-bit wide format and rounds it once to X80 with the current
// rounding mode and precision, raising the inexact exception and, if the
// result is out of range, overflow or underflow.  The evaluation error is far
// below the rounding error, so results are almost always correctly rounded.
// Results that are exact by definition, such as Exp(0) = 1, Log2(2^k) = k,
// Log10(10^k) = k for 0 <= k <= 27 or Cos(0) = 1, are returned exactly and
// without exceptions.

var (
	wLn2      = wideConst(ConstLn2)
	wLog2E    = wideConst(ConstLog2E)
	wLog10E   = wideConst(ConstLog10E)
	wLn10     = wideConst(ConstLn10)
	wLog10Of2 = wideConst(ConstLog10Of2)
	wPi       = wideConst(ConstPi)
	wHalfPi   = wPi.scale(-1)
	wPiOver4  = wPi.scale(-2)
)

// Series coefficients and tables, computed once from exact integer
// operations.
var (
	expCoeffs   []wide // 1/n!, n = 0..11, for e^x with |x| < 2^-8
	expm1Coeffs []wide // 1/(n+1)!, n = 0..10, for (e^x-1)/x with |x| < 2^-9
	lnCoeffs    []wide // 1/(2j+1), j = 0..9, for atanh(s)/s with |s| < 0.012
	atanCoeffs  []wide // (-1)^j/(2j+1), j = 0..11, for atan(t)/t with |t| < 1/32
	sinCoeffs   []wide // (-1)^j/(2j+1)!, j = 0..14, for sin(r)/r with |r| <= pi/4
	cosCoeffs   []wide // (-1)^j/(2j)!, j = 0..15, for cos(r) with |r| <= pi/4

	lnTable   [lnTableHi - lnTableLo + 1]wide // ln(i/32)
	atanTable [17]wide                        // atan(i/16)
)

const (
	expSquarings = 8
	lnTableLo    = 22 // ln table covers c = i/32 for m in [sqrt(1/2), sqrt(2))
	lnTableHi    = 46
)

func init() {
	fact := func(n int) wide { // 1/n!
		f := wOne
		for i := 2; i <= n; i++ {
			f = f.divInt(uint64(i))
		}
		return f
	}
	for n := range 12 {
		expCoeffs = append(expCoeffs, fact(n))
	}
	for n := range 11 {
		expm1Coeffs = append(expm1Coeffs, fact(n+1))
	}
	for j := range 30 {
		c := wOne.divInt(uint64(2*j + 1))
		if j < 10 {
			lnCoeffs = append(lnCoeffs, c)
		}
		if j < 12 {
			atanCoeffs = append(atanCoeffs, wide{j&1 != 0, c.exp, c.hi, c.lo})
		}
	}
	for j := range 16 {
		if j < 15 {
			s := fact(2*j + 1)
			sinCoeffs = append(sinCoeffs, wide{j&1 != 0, s.exp, s.hi, s.lo})
		}
		c := fact(2 * j)
		cosCoeffs = append(cosCoeffs, wide{j&1 != 0, c.exp, c.hi, c.lo})
	}

	// The tables are filled with long series that need no table themselves.
	var lnSlowCoeffs []wide
	for j := range 40 {
		lnSlowCoeffs = append(lnSlowCoeffs, wOne.divInt(uint64(2*j+1)))
	}
	for i := lnTableLo; i <= lnTableHi; i++ {
		c := wideFromInt(i).scale(-5)
		s := c.sub(wOne).div(c.add(wOne)) // |s| < 0.19
		lnTable[i-lnTableLo] = s.mul(horner(s.mul(s), lnSlowCoeffs)).scale(1)
	}
	var atanSlowCoeffs []wide
	for j := range 60 {
		c := wOne.divInt(uint64(2*j + 1))
		atanSlowCoeffs = append(atanSlowCoeffs, wide{j&1 != 0, c.exp, c.hi, c.lo})
	}
	for i := range atanTable {
		// atan(c) = 2*atan(c/(1+sqrt(1+c^2))) brings c below 0.42
		c := wideFromInt(i).scale(-4)
		t := c.div(wOne.add(wOne.add(c.mul(c)).sqrt()))
		atanTable[i] = t.mul(horner(t.mul(t), atanSlowCoeffs)).scale(1)
	}
}

// approx returns z, an approximation of the function value, rounded to the
// current RoundingPrecision, and raises the inexact exception once.
func approx(z X80) X80 {
	raised := 0
	if RoundingPrecision != 80 {
		raised = captureExceptions(func() { z = z.RoundToPrecision(RoundingPrecision) })
	}
	Raise(raised | ExceptionInexact)
	return z
}

// approxTiny returns a as the value of a function f with f(x) ~ x for tiny
// x, raising the inexact exception, and underflow if a is subnormal.
func approxTiny(a X80) X80 {
	if a.exp() == 0 {
		Raise(ExceptionUnderflow | ExceptionInexact)
		return a
	}
	return approx(a)
}

// isInteger reports whether the finite canonical value a is an integer.
func isInteger(a X80) bool {
	aSig, aExp := a.frac(), a.exp()
	switch {
	case aSig == 0 || aExp >= 0x403E:
		return true
	case aExp < 0x3FFF:
		return false
	}
	return aSig<<(aExp-0x3FFF+1) == 0
}

// absGtOne reports whether |a| > 1 for canonical a, including infinities.
func absGtOne(a X80) bool {
	return a.exp() > 0x3FFF || (a.exp() == 0x3FFF && a.frac() != 1<<63)
}

// isAbsOne reports whether |a| = 1 for canonical a.
func isAbsOne(a X80) bool {
	return a.exp() == 0x3FFF && a.frac() == 1<<63
}

// ---- logarithms ----

// lnParts splits the finite x > 0 into x = m*2^k with m in [sqrt(1/2),
// sqrt(2)) and returns k and ln(m).
func lnParts(x wide) (k int, lnM wide) {
	k = x.exp
	m := x
	m.exp = 0
	if m.hi >= 0xB504F333F9DE6484 { // m >= sqrt(2)
		m.exp, k = -1, k+1
	}
	// ln(m) = ln(c) + 2*atanh(s), c = i/32 nearest to m, s = (m-c)/(m+c)
	i := int(m.scale(5).float64() + 0.5)
	c := wideFromInt(i).scale(-5)
	s := m.sub(c).div(m.add(c))
	return k, lnTable[i-lnTableLo].add(s.mul(horner(s.mul(s), lnCoeffs)).scale(1))
}

// lnW returns ln(x) for finite x > 0.
func lnW(x wide) wide {
	k, lnM := lnParts(x)
	return wideFromInt(k).mul(wLn2).add(lnM)
}

// logSpecial handles the operands of a logarithm that need no approximation.
func logSpecial(a X80) (z X80, done bool) {
	aSig, aExp, aSign := a.frac(), a.exp(), a.sign()
	switch {
	case aExp == 0x7FFF && aSig<<1 != 0:
		return propagateFloatX80NaN(a, a), true
	case aExp == 0 && aSig == 0:
		Raise(ExceptionDivbyzero)
		return X80InfNeg, true
	case aSign:
		Raise(ExceptionInvalid)
		return DefaultNaN, true
	case aExp == 0x7FFF:
		return a, true
	case aExp == 0x3FFF && aSig == 1<<63:
		return X80Zero, true // log(1) = 0 exactly
	}
	return X80{}, false
}

// Ln returns the natural logarithm of `a'.  Ln(+Inf) = +Inf, Ln(±0) = -Inf
// with the divide-by-zero exception, and Ln of a negative value is NaN with
// the invalid exception.  This is the 68881/68882 FLOGN operation.
func (a X80) Ln() X80 {
	a = a.canonical()
	if z, done := logSpecial(a); done {
		return z
	}
	return lnW(wideFromX80(a)).round()
}

// Log2 returns the binary logarithm of `a', with the special cases of Ln.
// Log2(2^k) = k exactly.  This is the 68881/68882 FLOG2 operation.
func (a X80) Log2() X80 {
	a = a.canonical()
	if z, done := logSpecial(a); done {
		return z
	}
	x := wideFromX80(a)
	if x.hi == 1<<63 && x.lo == 0 {
		return intX80(x.exp)
	}
	k, lnM := lnParts(x)
	return wideFromInt(k).add(lnM.mul(wLog2E)).round()
}

// Log10 returns the decimal logarithm of `a', with the special cases of Ln.
// Log10(10^k) = k exactly for 0 <= k <= 27, the powers of ten that are exact
// X80 values.  This is the 68881/68882 FLOG10 operation.
func (a X80) Log10() X80 {
	a = a.canonical()
	if z, done := logSpecial(a); done {
		return z
	}
	k, lnM := lnParts(wideFromX80(a))
	z := wideFromInt(k).mul(wLog10Of2).add(lnM.mul(wLog10E))
	if n := int(math.Round(z.float64())); 0 <= n && n < len(pow10Exact) && a == pow10Exact[n] {
		return intX80(n)
	}
	return z.round()
}

// Log1p returns ln(1+a), accurate also for `a' near zero.  Log1p(-1) = -Inf
// with the divide-by-zero exception, and Log1p(a) for a < -1 is NaN with the
// invalid exception.  This is the 68881/68882 FLOGNP1 operation.
func (a X80) Log1p() X80 {
	a = a.canonical()
	aSig, aExp, aSign := a.frac(), a.exp(), a.sign()
	switch {
	case aExp == 0x7FFF && aSig<<1 != 0:
		return propagateFloatX80NaN(a, a)
	case aSign && isAbsOne(a):
		Raise(ExceptionDivbyzero)
		return X80InfNeg
	case aSign && absGtOne(a):
		Raise(ExceptionInvalid)
		return DefaultNaN
	case aExp == 0x7FFF || aSig == 0:
		return a
	case aExp < 0x3FFF-64: // ln(1+x) = x - x^2/2 + ... rounds to x
		return approxTiny(a)
	}
	// 1+x is exact in the wide format for |x| >= 2^-64, and ln(m) for m near
	// 1 is computed without cancellation
	return lnW(wOne.add(wideFromX80(a))).round()
}

// ---- exponentials ----

// expW returns e^x for |x| < 2^15.
func expW(x wide) wide {
	k := int(math.Round(x.float64() * math.Log2E))
	r := x.sub(wideFromInt(k).mul(wLn2)) // |r| <= ln(2)/2
	// e^r = (e^(r/2^8))^(2^8)
	y := horner(r.scale(-expSquarings), expCoeffs)
	for range expSquarings {
		y = y.mul(y)
	}
	return y.scale(k)
}

// expm1W returns e^x - 1 for |x| < 2^7.
func expm1W(x wide) wide {
	if x.exp >= -2 { // |x| >= 1/4: e^x - 1 has no cancellation
		return expW(x).sub(wOne)
	}
	// e^(2y) - 1 = (e^y - 1)(e^y - 1 + 2), from |y| < 2^-9
	s := max(x.exp+10, 0)
	y := x.scale(-s)
	e := y.mul(horner(y, expm1Coeffs))
	for range s {
		e = e.mul(e.add(wTwo))
	}
	return e
}

// expSpecial handles the operands of an exponential that need no
// approximation; limitExp is the biased exponent from which |a| certainly
// overflows or underflows.
func expSpecial(a X80, limitExp int) (z X80, done bool) {
	aSig, aExp, aSign := a.frac(), a.exp(), a.sign()
	switch {
	case aExp == 0x7FFF && aSig<<1 != 0:
		return propagateFloatX80NaN(a, a), true
	case aExp == 0x7FFF && aSign:
		return X80Zero, true
	case aExp == 0x7FFF:
		return a, true
	case aSig == 0:
		return X80One, true
	case aExp >= limitExp && aSign:
		return wOne.scale(-0x10000).round(), true
	case aExp >= limitExp:
		return wOne.scale(0x10000).round(), true
	}
	return X80{}, false
}

// Exp returns e^a.  Exp(+Inf) = +Inf, Exp(-Inf) = +0 and Exp(±0) = 1;
// results out of range overflow or underflow.  This is the 68881/68882 FETOX
// operation.
func (a X80) Exp() X80 {
	a = a.canonical()
	if z, done := expSpecial(a, 0x3FFF+14); done { // |a| >= 16384
		return z
	}
	return expW(wideFromX80(a)).round()
}

// Exp2 returns 2^a, with the special cases of Exp.  Exp2(n) is exact for
// integers n in range.  This is the 68881/68882 FTWOTOX operation.
func (a X80) Exp2() X80 {
	a = a.canonical()
	if z, done := expSpecial(a, 0x3FFF+15); done { // |a| >= 32768
		return z
	}
	if isInteger(a) {
		return X80One.Scale(int(a.ToInt64()))
	}
	return expW(wideFromX80(a).mul(wLn2)).round()
}

// Exp10 returns 10^a, with the special cases of Exp.  Exp10(n) equals
// Pow10(n) for integers n and is exact for 0 <= n <= 27.  This is the
// 68881/68882 FTENTOX operation.
func (a X80) Exp10() X80 {
	a = a.canonical()
	if z, done := expSpecial(a, 0x3FFF+13); done { // |a| >= 8192
		return z
	}
	if isInteger(a) {
		return Pow10(int(a.ToInt64()))
	}
	return expW(wideFromX80(a).mul(wLn10)).round()
}

// Expm1 returns e^a - 1, accurate also for `a' near zero.  Expm1(+Inf) = +Inf
// and Expm1(-Inf) = -1.  This is the 68881/68882 FETOXM1 operation.
func (a X80) Expm1() X80 {
	a = a.canonical()
	aSig, aExp, aSign := a.frac(), a.exp(), a.sign()
	switch {
	case aExp == 0x7FFF && aSig<<1 != 0:
		return propagateFloatX80NaN(a, a)
	case aExp == 0x7FFF && aSign:
		return X80MinusOne
	case aExp == 0x7FFF || aSig == 0:
		return a
	case aExp < 0x3FFF-64: // e^x - 1 = x + x^2/2 + ... rounds to x
		return approxTiny(a)
	case aExp >= 0x3FFF+6 && !aSign: // x >= 64: e^x - 1 rounds to e^x
		return a.Exp()
	case aExp >= 0x3FFF+6: // x <= -64: e^x - 1 rounds to -1
		return approx(X80MinusOne)
	}
	return expm1W(wideFromX80(a)).round()
}

// ---- hyperbolic functions ----

// Sinh returns the hyperbolic sine of `a'.  Sinh(±0) = ±0 and
// Sinh(±Inf) = ±Inf.  This is the 68881/68882 FSINH operation.
func (a X80) Sinh() X80 {
	a = a.canonical()
	aSig, aExp := a.frac(), a.exp()
	switch {
	case aExp == 0x7FFF && aSig<<1 != 0:
		return propagateFloatX80NaN(a, a)
	case aExp == 0x7FFF || aSig == 0:
		return a
	case aExp < 0x3FFF-32: // sinh(x) = x + x^3/6 + ... rounds to x
		return approxTiny(a)
	}
	x := wideFromX80(a)
	var z wide
	switch {
	case aExp >= 0x3FFF+14: // overflows
		z = wOne.scale(0x10000)
	case aExp >= 0x3FFF+5: // |x| >= 32: sinh(x) = e^|x|/2
		z = expW(x.abs()).scale(-1)
	default: // sinh(x) = (t + t/(t+1))/2 with t = e^|x| - 1
		t := expm1W(x.abs())
		z = t.add(t.div(t.add(wOne))).scale(-1)
	}
	z.neg = x.neg
	return z.round()
}

// Cosh returns the hyperbolic cosine of `a'.  Cosh(±0) = 1 and
// Cosh(±Inf) = +Inf.  This is the 68881/68882 FCOSH operation.
func (a X80) Cosh() X80 {
	a = a.canonical()
	aSig, aExp := a.frac(), a.exp()
	switch {
	case aExp == 0x7FFF && aSig<<1 != 0:
		return propagateFloatX80NaN(a, a)
	case aExp == 0x7FFF:
		return a.Abs()
	case aSig == 0:
		return X80One
	case aExp < 0x3FFF-32: // cosh(x) = 1 + x^2/2 + ... rounds to 1
		return approx(X80One)
	case aExp >= 0x3FFF+14: // overflows
		return wOne.scale(0x10000).round()
	}
	x := wideFromX80(a).abs()
	if aExp >= 0x3FFF+5 { // |x| >= 32: cosh(x) = e^|x|/2
		return expW(x).scale(-1).round()
	}
	e := expW(x)
	return e.add(e.recip()).scale(-1).round()
}

// Tanh returns the hyperbolic tangent of `a'.  Tanh(±0) = ±0 and
// Tanh(±Inf) = ±1.  This is the 68881/68882 FTANH operation.
func (a X80) Tanh() X80 {
	a = a.canonical()
	aSig, aExp, aSign := a.frac(), a.exp(), a.sign()
	one := X80One
	if aSign {
		one = X80MinusOne
	}
	switch {
	case aExp == 0x7FFF && aSig<<1 != 0:
		return propagateFloatX80NaN(a, a)
	case aExp == 0x7FFF:
		return one
	case aSig == 0:
		return a
	case aExp < 0x3FFF-32: // tanh(x) = x - x^3/3 + ... rounds to x
		return approxTiny(a)
	case aExp >= 0x3FFF+5: // |x| >= 32: tanh(x) rounds to ±1
		return approx(one)
	}
	// tanh(x) = t/(t+2) with t = e^(2|x|) - 1
	t := expm1W(wideFromX80(a).abs().scale(1))
	z := t.div(t.add(wTwo))
	z.neg = aSign
	return z.round()
}

// Atanh returns the inverse hyperbolic tangent of `a'.  Atanh(±1) = ±Inf with
// the divide-by-zero exception, and Atanh(a) for |a| > 1 is NaN with the
// invalid exception.  This is the 68881/68882 FATANH operation.
func (a X80) Atanh() X80 {
	a = a.canonical()
	aSig, aExp, aSign := a.frac(), a.exp(), a.sign()
	switch {
	case aExp == 0x7FFF && aSig<<1 != 0:
		return propagateFloatX80NaN(a, a)
	case absGtOne(a):
		Raise(ExceptionInvalid)
		return DefaultNaN
	case isAbsOne(a):
		Raise(ExceptionDivbyzero)
		return packFloatX80(aSign, 0x7FFF, 1<<63)
	case aSig == 0:
		return a
	case aExp < 0x3FFF-32: // atanh(x) = x + x^3/3 + ... rounds to x
		return approxTiny(a)
	}
	// atanh(x) = ln((1+x)/(1-x))/2, with 1±x exact in the wide format
	x := wideFromX80(a).abs()
	z := lnW(wOne.add(x).div(wOne.sub(x))).scale(-1)
	z.neg = aSign
	return z.round()
}

// ---- inverse trigonometric functions ----

// atanW returns atan(x) for finite x >= 0.
func atanW(x wide) wide {
	invert := x.cmpAbs(wOne) > 0
	if invert {
		x = x.recip() // atan(x) = pi/2 - atan(1/x)
	}
	// atan(x) = atan(c) + atan(t), c = i/16 nearest to x, t = (x-c)/(1+x*c)
	i := int(x.scale(4).float64() + 0.5)
	c := wideFromInt(i).scale(-4)
	t := x.sub(c).div(wOne.add(x.mul(c)))
	z := atanTable[i].add(t.mul(horner(t.mul(t), atanCoeffs)))
	if invert {
		z = wHalfPi.sub(z)
	}
	return z
}

// Atan returns the arctangent, in radians, of `a'.  Atan(±0) = ±0 and
// Atan(±Inf) = ±pi/2.  This is the 68881/68882 FATAN operation.
func (a X80) Atan() X80 {
	a = a.canonical()
	aSig, aExp, aSign := a.frac(), a.exp(), a.sign()
	switch {
	case aExp == 0x7FFF && aSig<<1 != 0:
		return propagateFloatX80NaN(a, a)
	case aSig == 0:
		return a
	case aExp < 0x3FFF-32: // atan(x) = x - x^3/3 + ... rounds to x
		return approxTiny(a)
	}
	z := wHalfPi
	if aExp != 0x7FFF {
		z = atanW(wideFromX80(a).abs())
	}
	z.neg = aSign
	return z.round()
}

// Asin returns the arcsine, in radians, of `a'.  Asin(±0) = ±0, and Asin(a)
// for |a| > 1 is NaN with the invalid exception.  This is the 68881/68882
// FASIN operation.
func (a X80) Asin() X80 {
	a = a.canonical()
	aSig, aExp, aSign := a.frac(), a.exp(), a.sign()
	switch {
	case aExp == 0x7FFF && aSig<<1 != 0:
		return propagateFloatX80NaN(a, a)
	case absGtOne(a):
		Raise(ExceptionInvalid)
		return DefaultNaN
	case aSig == 0:
		return a
	case aExp < 0x3FFF-32: // asin(x) = x + x^3/6 + ... rounds to x
		return approxTiny(a)
	}
	z := wHalfPi
	if !isAbsOne(a) {
		// asin(x) = atan(x/sqrt((1-x)(1+x))), with 1±x exact
		x := wideFromX80(a).abs()
		z = atanW(x.div(wOne.sub(x).mul(wOne.add(x)).sqrt()))
	}
	z.neg = aSign
	return z.round()
}

// Acos returns the arccosine, in radians, of `a'.  Acos(1) = +0, and Acos(a)
// for |a| > 1 is NaN with the invalid exception.  This is the 68881/68882
// FACOS operation.
func (a X80) Acos() X80 {
	a = a.canonical()
	aSig, aExp, aSign := a.frac(), a.exp(), a.sign()
	switch {
	case aExp == 0x7FFF && aSig<<1 != 0:
		return propagateFloatX80NaN(a, a)
	case absGtOne(a):
		Raise(ExceptionInvalid)
		return DefaultNaN
	case isAbsOne(a) && !aSign:
		return X80Zero
	case isAbsOne(a):
		return wPi.round()
	}
	// acos(x) = 2*atan(sqrt((1-x)/(1+x))), with 1±x exact
	x := wideFromX80(a)
	return atanW(wOne.sub(x).div(wOne.add(x)).sqrt()).scale(1).round()
}

// ---- trigonometric functions ----

// trigReduce returns n in [0, 3] and r with a = (4j+n)*pi/2 + r for some
// integer j and |r| <= pi/4, for finite canonical a.  Arguments beyond pi/4
// are reduced with the Payne-Hanek method: the significand is multiplied by
// a window of the bits of 2/pi wide enough that r is accurate for every X80
// value.
func trigReduce(a X80) (int, wide) {
	x := wideFromX80(a)
	if x.cmpAbs(wPiOver4) <= 0 {
		return 0, x
	}
	// |a| = sig * 2^e; the words of 2/pi before index i0 only contribute
	// multiples of 4 to |a|*2/pi, which leave the quadrant unchanged
	sig, e := a.frac(), a.exp()-0x3FFF-63
	i0 := 0
	if e >= 2 {
		i0 = (e - 2) >> 6
	}
	// q = sig * twoOverPi[i0:i0+6], least significant word first
	var q [7]uint64
	var carry uint64
	for j := 5; j >= 0; j-- {
		hi, lo := bits.Mul64(sig, twoOverPi[i0+j])
		var c uint64
		q[5-j], c = bits.Add64(lo, carry, 0)
		carry = hi + c
	}
	q[6] = carry
	// |a|*2/pi = q * 2^-b (mod 4); take the quadrant and 192 fraction bits
	b := 64*(i0+6) - e
	n := int(bitsAt(&q, b) & 3)
	f2, f1, f0 := bitsAt(&q, b-64), bitsAt(&q, b-128), bitsAt(&q, b-192)
	neg := false
	if f2>>63 != 0 { // fraction >= 1/2: round the quotient up
		n = (n + 1) & 3
		var borrow uint64
		f0, borrow = bits.Sub64(0, f0, 0)
		f1, borrow = bits.Sub64(0, f1, borrow)
		f2, _ = bits.Sub64(0, f2, borrow)
		neg = true
	}
	// fraction = f2:f1:f0 * 2^-192
	exp := -1
	for f2 == 0 && (f1|f0) != 0 {
		f2, f1, f0, exp = f1, f0, 0, exp-64
	}
	if s := bits.LeadingZeros64(f2); s != 0 && s != 64 {
		f2, f1, exp = f2<<s|f1>>(64-s), f1<<s|f0>>(64-s), exp-s
	}
	r := wide{neg, exp, f2, f1}.mul(wHalfPi)
	if f2 == 0 {
		r = wide{}
	}
	if a.sign() {
		n, r.neg = (4-n)&3, !r.neg
	}
	return n, r
}

// bitsAt returns the 64 bits of q starting at bit pos, counted from the least
// significant bit; bits outside q are zero.
func bitsAt(q *[7]uint64, pos int) uint64 {
	w, s := pos>>6, uint(pos&63)
	var lo, hi uint64
	if w >= 0 && w < len(q) {
		lo = q[w]
	}
	if w+1 >= 0 && w+1 < len(q) {
		hi = q[w+1]
	}
	if s == 0 {
		return lo
	}
	return lo>>s | hi<<(64-s)
}

// sinCosW returns sin(r) and cos(r) for |r| <= pi/4.
func sinCosW(r wide) (sin, cos wide) {
	r2 := r.mul(r)
	return r.mul(horner(r2, sinCoeffs)), horner(r2, cosCoeffs)
}

// sinCosSelect returns sin(n*pi/2 + r) for |r| <= pi/4, evaluating only the
// series it needs.
func sinCosSelect(n int, r wide) wide {
	var z wide
	if n&1 == 0 {
		z = r.mul(horner(r.mul(r), sinCoeffs))
	} else {
		z = horner(r.mul(r), cosCoeffs)
	}
	if n&2 != 0 {
		z = z.negate()
	}
	return z
}

// sinCosQuadrant returns sin(a) and cos(a) for finite canonical a.
func sinCosQuadrant(a X80) (sin, cos wide) {
	n, r := trigReduce(a)
	s, c := sinCosW(r)
	switch n {
	case 1:
		return c, s.negate()
	case 2:
		return s.negate(), c.negate()
	case 3:
		return c.negate(), s
	}
	return s, c
}

// trigSpecial handles NaN and infinite operands of the trigonometric
// functions.
func trigSpecial(a X80) (z X80, done bool) {
	if a.exp() != 0x7FFF {
		return X80{}, false
	}
	if a.frac()<<1 != 0 {
		return propagateFloatX80NaN(a, a), true
	}
	Raise(ExceptionInvalid)
	return DefaultNaN, true
}

// Sin returns the sine of the radian argument `a'.  Sin(±0) = ±0, and
// Sin(±Inf) is NaN with the invalid exception.  The argument reduction is
// accurate for all finite arguments.  This is the 68881/68882 FSIN operation.
func (a X80) Sin() X80 {
	a = a.canonical()
	if z, done := trigSpecial(a); done {
		return z
	}
	switch {
	case a.frac() == 0:
		return a
	case a.exp() < 0x3FFF-32: // sin(x) = x - x^3/6 + ... rounds to x
		return approxTiny(a)
	}
	n, r := trigReduce(a)
	return sinCosSelect(n, r).round()
}

// Cos returns the cosine of the radian argument `a'.  Cos(±0) = 1, and
// Cos(±Inf) is NaN with the invalid exception.  This is the 68881/68882 FCOS
// operation.
func (a X80) Cos() X80 {
	a = a.canonical()
	if z, done := trigSpecial(a); done {
		return z
	}
	switch {
	case a.frac() == 0:
		return X80One
	case a.exp() < 0x3FFF-32: // cos(x) = 1 - x^2/2 + ... rounds to 1
		return approx(X80One)
	}
	n, r := trigReduce(a)
	return sinCosSelect(n+1, r).round() // cos(x) = sin(x + pi/2)
}

// Sincos returns Sin(a) and Cos(a), computed together.  This is the
// 68881/68882 FSINCOS operation.
func (a X80) Sincos() (sin, cos X80) {
	a = a.canonical()
	if z, done := trigSpecial(a); done {
		return z, z
	}
	switch {
	case a.frac() == 0:
		return a, X80One
	case a.exp() < 0x3FFF-32:
		return approxTiny(a), X80One
	}
	s, c := sinCosQuadrant(a)
	// both results are inexact; report the exceptions once
	raised := captureExceptions(func() { sin, cos = s.round(), c.round() })
	Raise(raised)
	return sin, cos
}

// Tan returns the tangent of the radian argument `a'.  Tan(±0) = ±0, and
// Tan(±Inf) is NaN with the invalid exception.  This is the 68881/68882 FTAN
// operation.
func (a X80) Tan() X80 {
	a = a.canonical()
	if z, done := trigSpecial(a); done {
		return z
	}
	switch {
	case a.frac() == 0:
		return a
	case a.exp() < 0x3FFF-32: // tan(x) = x + x^3/3 + ... rounds to x
		return approxTiny(a)
	}
	n, r := trigReduce(a)
	s, c := sinCosW(r)
	if n&1 != 0 {
		return c.div(s).negate().round()
	}
	return s.div(c).round()
}

func intX80(n int) X80 {
	return Int64ToFloatX80(int64(n))
}
