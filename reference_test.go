package float

// Differential tests against math/big, which rounds exactly. Random operands
// cover the whole significand and a wide exponent range; results outside the
// normal X80 range are skipped because big.Float has no subnormals.

import (
	"encoding/binary"
	"math"
	"math/big"
	"math/rand/v2"
	"testing"
)

const iterations = 50000

func toBig(a X80) *big.Float {
	z := new(big.Float).SetPrec(64)
	switch {
	case a.IsNaN():
		return nil
	case a.IsInf():
		return z.SetInf(a.sign())
	}
	exp := a.exp()
	if exp == 0 {
		exp = 1
	}
	z.SetUint64(a.frac()).SetMantExp(z, exp-0x3FFF-63)
	if a.sign() {
		z.Neg(z)
	}
	return z
}

// inNormalRange reports whether x is representable as a normal X80 value.
func inNormalRange(x *big.Float) bool {
	if x.IsInf() || x.Sign() == 0 {
		return x.Sign() == 0
	}
	e := x.MantExp(nil) - 1 // x = 1.f * 2^e
	return e > -0x3FFE && e < 0x4000
}

// randX80 returns a normal value with a random significand and an exponent
// within ±spread of 2^0.
func randX80(r *rand.Rand, spread int) X80 {
	return packFloatX80(r.IntN(2) == 0, 0x3FFF+r.IntN(2*spread+1)-spread, r.Uint64()|1<<63)
}

func bigOf(prec uint) *big.Float {
	return new(big.Float).SetPrec(prec).SetMode(big.ToNearestEven)
}

func mustParse(s string) *big.Float {
	f, _, err := big.ParseFloat(s, 10, 300, big.ToNearestEven)
	if err != nil {
		panic(err)
	}
	return f
}

func TestReferenceArithmetic(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	ops := []struct {
		name string
		x80  func(a, b X80) X80
		ref  func(a, b *big.Float) *big.Float
	}{
		{"Add", X80.Add, func(a, b *big.Float) *big.Float { return bigOf(64).Add(a, b) }},
		{"Sub", X80.Sub, func(a, b *big.Float) *big.Float { return bigOf(64).Sub(a, b) }},
		{"Mul", X80.Mul, func(a, b *big.Float) *big.Float { return bigOf(64).Mul(a, b) }},
		{"Div", X80.Div, func(a, b *big.Float) *big.Float { return bigOf(64).Quo(a, b) }},
	}
	for _, op := range ops {
		t.Run(op.name, func(t *testing.T) {
			fails := 0
			for i := range iterations {
				// alternate between close exponents (cancellation, carries)
				// and far apart ones (alignment shifts, sticky bits)
				spread := 2
				if i%2 == 1 {
					spread = 100
				}
				a, b := randX80(r, spread), randX80(r, spread)
				want := op.ref(toBig(a), toBig(b))
				if !inNormalRange(want) {
					continue
				}
				if got := op.x80(a, b); toBig(got).Cmp(want) != 0 {
					t.Errorf("%s(%s, %s) = %s, want %s", op.name, a.Internal(), b.Internal(), got.Internal(), want.Text('p', 0))
					if fails++; fails > 5 {
						t.FailNow()
					}
				}
			}
		})
	}
}

func TestReferenceSqrt(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 4))
	for range iterations {
		a := randX80(r, 1000)
		a.high &^= 0x8000
		// a halfway case would need sqrt(a) to have exactly 65 significant
		// bits, which is impossible for a 64-bit a, so rounding twice is safe
		want := bigOf(64).Set(bigOf(200).Sqrt(toBig(a)))
		if got := a.Sqrt(); toBig(got).Cmp(want) != 0 {
			t.Fatalf("Sqrt(%s) = %s, want %s", a.Internal(), got.Internal(), want.Text('p', 0))
		}
	}
}

// remRef computes the IEEE remainder a - n*b, n = round-half-even(a/b), exactly.
func remRef(a, b X80) *big.Float {
	ab, bb := toBig(a), toBig(b)
	am, bm := new(big.Int), new(big.Int)
	ae := ab.MantExp(nil) - 64
	be := bb.MantExp(nil) - 64
	new(big.Float).SetMantExp(ab, -ae).Int(am)
	new(big.Float).SetMantExp(bb, -be).Int(bm)
	// align both to the smaller exponent
	e := min(ae, be)
	am.Lsh(am, uint(ae-e))
	bm.Lsh(bm, uint(be-e))
	n, rem := new(big.Int).QuoRem(am, bm, new(big.Int))
	twice := new(big.Int).Abs(new(big.Int).Lsh(rem, 1))
	if c := twice.CmpAbs(bm); c > 0 || (c == 0 && n.Bit(0) == 1) {
		if rem.Sign() == bm.Sign() {
			rem.Sub(rem, bm)
		} else {
			rem.Add(rem, bm)
		}
	}
	z := new(big.Float).SetInt(rem)
	z.SetMantExp(z, e)
	if z.Sign() == 0 && a.sign() {
		z.Neg(z)
	}
	return z
}

func TestReferenceRem(t *testing.T) {
	r := rand.New(rand.NewPCG(5, 6))
	for i := range iterations {
		a, b := randX80(r, 150), randX80(r, 20)
		if i%4 == 0 {
			// small integers produce exact ties such as 7 rem 2
			a = Int64ToFloatX80(r.Int64N(2001) - 1000)
			b = Int64ToFloatX80(r.Int64N(20) + 1)
		}
		want := remRef(a, b)
		if got := a.Rem(b); toBig(got).Cmp(want) != 0 {
			t.Fatalf("Rem(%s, %s) = %s, want %s", a.Internal(), b.Internal(), got.Internal(), want.Text('g', 25))
		}
	}
}

func TestReferenceCompare(t *testing.T) {
	r := rand.New(rand.NewPCG(7, 8))
	specials := []X80{X80Zero, packFloatX80(true, 0, 0), {0, 1}, X80InfPos, X80InfNeg, X80One, X80MinusOne}
	for i := range iterations {
		a, b := randX80(r, 3), randX80(r, 3)
		switch i % 4 {
		case 0:
			b = a
		case 1:
			a = specials[r.IntN(len(specials))]
		case 2:
			a, b = specials[r.IntN(len(specials))], specials[r.IntN(len(specials))]
		}
		c := toBig(a).Cmp(toBig(b))
		checks := []struct {
			name      string
			got, want bool
		}{
			{"Eq", a.Eq(b), c == 0}, {"EqSignaling", a.EqSignaling(b), c == 0},
			{"Lt", a.Lt(b), c < 0}, {"LtQuiet", a.LtQuiet(b), c < 0},
			{"Le", a.Le(b), c <= 0}, {"LeQuiet", a.LeQuiet(b), c <= 0},
			{"Gt", a.Gt(b), c > 0}, {"GtQuiet", a.GtQuiet(b), c > 0},
			{"Ge", a.Ge(b), c >= 0}, {"GeQuiet", a.GeQuiet(b), c >= 0},
		}
		for _, ch := range checks {
			if ch.got != ch.want {
				t.Fatalf("%s.%s(%s) = %v, want %v", a.Internal(), ch.name, b.Internal(), ch.got, ch.want)
			}
		}
	}
}

func TestCompareNaNIsUnordered(t *testing.T) {
	for _, f := range []func(a, b X80) bool{X80.Eq, X80.Lt, X80.Le, X80.Gt, X80.Ge, X80.LtQuiet, X80.LeQuiet, X80.GtQuiet, X80.GeQuiet} {
		if f(X80NaN, X80One) || f(X80One, X80NaN) || f(X80NaN, X80NaN) {
			t.Fatal("comparison involving NaN returned true")
		}
	}
}

func TestReferenceToFloat(t *testing.T) {
	r := rand.New(rand.NewPCG(9, 10))
	for _, mode := range []int{RoundNearestEven, RoundToZero, RoundDown, RoundUp} {
		RoundingMode = mode
		bigMode := map[int]big.RoundingMode{RoundNearestEven: big.ToNearestEven, RoundToZero: big.ToZero, RoundDown: big.ToNegativeInf, RoundUp: big.ToPositiveInf}[mode]
		for range iterations {
			// covers float64 and float32 subnormals and overflow
			a := randX80(r, 1100)
			if r.IntN(2) == 0 {
				a = randX80(r, 160)
			}
			// big.Float.Float64 and Float32 always round to nearest, so for the
			// directed modes round explicitly and compare normal results only
			if want, ok := roundTo(toBig(a), 53, bigMode, -1022, 1023, mode); ok {
				if got := a.ToFloat64(); math.Float64bits(got) != math.Float64bits(want) {
					t.Fatalf("mode %d: ToFloat64(%s) = %v, want %v", mode, a.Internal(), got, want)
				}
			}
			if want, ok := roundTo(toBig(a), 24, bigMode, -126, 127, mode); ok {
				if got := a.ToFloat32(); math.Float32bits(got) != math.Float32bits(float32(want)) {
					t.Fatalf("mode %d: ToFloat32(%s) = %v, want %v", mode, a.Internal(), got, want)
				}
			}
		}
	}
	RoundingMode = RoundNearestEven
}

// roundTo rounds x to prec bits. In round-to-nearest mode it defers to
// big.Float.Float64, which also handles subnormals and overflow; otherwise it
// reports ok only if the result is normal, i.e. its exponent is in [minExp, maxExp].
func roundTo(x *big.Float, prec uint, mode big.RoundingMode, minExp, maxExp, x80Mode int) (float64, bool) {
	if x80Mode == RoundNearestEven {
		if prec == 24 {
			f, _ := x.Float32()
			return float64(f), true
		}
		f, _ := x.Float64()
		return f, true
	}
	r := new(big.Float).SetPrec(prec).SetMode(mode).Set(x)
	if e := r.MantExp(nil) - 1; e < minExp || e > maxExp {
		return 0, false
	}
	f, _ := r.Float64()
	return f, true
}

func TestToFloatOverflowDirected(t *testing.T) {
	defer func() { RoundingMode = RoundNearestEven }()
	huge := packFloatX80(false, 0x3FFF+2000, 1<<63)
	RoundingMode = RoundToZero
	if got := huge.ToFloat64(); got != math.MaxFloat64 {
		t.Errorf("ToFloat64 round-to-zero overflow = %v, want MaxFloat64", got)
	}
	if got := huge.ToFloat32(); got != math.MaxFloat32 {
		t.Errorf("ToFloat32 round-to-zero overflow = %v, want MaxFloat32", got)
	}
	RoundingMode = RoundNearestEven
	if got := (X80{0x3FFF, 0x8000000000000400}).ToFloat64(); got != 1 {
		t.Errorf("ToFloat64(1+2^-53) = %v, want 1 (tie to even)", got)
	}
	if got := (X80{0x3FFF, 0x8000000000000C00}).ToFloat64(); got != 1+0x1p-51 {
		t.Errorf("ToFloat64(1+3*2^-53) = %v, want 1+2^-51 (tie to even)", got)
	}
}

func roundHalfEven(a X80, trunc bool) (*big.Int, bool) {
	b := toBig(a)
	n, _ := b.Int(nil) // truncates toward zero
	if trunc {
		return n, b.IsInt()
	}
	frac := new(big.Float).Sub(b, new(big.Float).SetInt(n))
	frac.Abs(frac)
	if c := frac.Cmp(big.NewFloat(0.5)); c > 0 || (c == 0 && n.Bit(0) == 1) {
		n.Add(n, big.NewInt(int64(b.Sign())))
	}
	return n, b.IsInt()
}

func TestReferenceToInt(t *testing.T) {
	r := rand.New(rand.NewPCG(11, 12))
	clamp := func(n *big.Int, lo, hi int64) (int64, bool) {
		if n.IsInt64() && n.Int64() >= lo && n.Int64() <= hi {
			return n.Int64(), false
		}
		if n.Sign() < 0 {
			return lo, true
		}
		return hi, true
	}
	for i := range iterations {
		a := randX80(r, 70)
		if i%8 == 0 {
			a = Int64ToFloatX80(r.Int64N(9) - 4).Div(Int64ToFloatX80(2)) // exact halves
		}
		for _, c := range []struct {
			name   string
			got    int64
			trunc  bool
			lo, hi int64
		}{
			{"ToInt64", a.ToInt64(), false, math.MinInt64, math.MaxInt64},
			{"ToInt64RoundZero", a.ToInt64RoundZero(), true, math.MinInt64, math.MaxInt64},
			{"ToInt32", int64(a.ToInt32()), false, math.MinInt32, math.MaxInt32},
			{"ToInt32RoundZero", int64(a.ToInt32RoundZero()), true, math.MinInt32, math.MaxInt32},
		} {
			n, _ := roundHalfEven(a, c.trunc)
			want, _ := clamp(n, c.lo, c.hi)
			if c.got != want {
				t.Fatalf("%s(%s) = %d, want %d", c.name, a.Internal(), c.got, want)
			}
		}
	}
	ClearExceptions()
	if got := packFloatX80(true, 0x403E, 1<<63).ToInt64RoundZero(); got != math.MinInt64 || HasException(ExceptionInvalid) {
		t.Errorf("ToInt64RoundZero(-2^63) = %d, invalid=%v; want MinInt64 without invalid", got, HasException(ExceptionInvalid))
	}
}

func TestReferenceFormat(t *testing.T) {
	r := rand.New(rand.NewPCG(13, 14))
	for i := range iterations / 5 {
		// extreme exponents need hundreds of decimal shifts each, so only
		// every 20th value spans the full range
		a := randX80(r, 80)
		if i%20 == 0 {
			a = randX80(r, 16000)
		}
		b := toBig(a)
		for _, c := range []struct {
			fmt  byte
			prec int
		}{{'e', 18}, {'E', 5}, {'g', 21}, {'g', 6}, {'f', 3}} {
			if c.fmt == 'f' && a.exp() > 0x3FFF+80 {
				continue
			}
			if got, want := a.Format(c.fmt, c.prec), b.Text(c.fmt, c.prec); got != want {
				t.Fatalf("Format(%s, %q, %d) = %s, want %s", a.Internal(), c.fmt, c.prec, got, want)
			}
		}
	}
}

func TestFormatSpecials(t *testing.T) {
	for _, c := range []struct {
		a    X80
		fmt  byte
		prec int
		want string
	}{
		{X80InfPos, 'e', 3, "+Inf"},
		{X80InfNeg, 'g', 3, "-Inf"},
		{X80NaN, 'f', 3, "NaN"},
		{packFloatX80(false, 0x3FFF+4000, 1<<63), 'e', 3, "1.318e+1204"},
		{packFloatX80(false, 0x3FFF-16000, 1<<63), 'e', 3, "3.312e-4817"},
		{X80Zero, 'g', -1, "0"},
	} {
		if got := c.a.Format(c.fmt, c.prec); got != c.want {
			t.Errorf("Format(%s, %q, %d) = %q, want %q", c.a.Internal(), c.fmt, c.prec, got, c.want)
		}
	}
}

func TestConstants(t *testing.T) {
	sqrt2 := bigOf(300).Sqrt(bigOf(300).SetInt64(2))
	for _, c := range []struct {
		name string
		got  X80
		want *big.Float
	}{
		{"Pi", X80Pi, mustParse("3.14159265358979323846264338327950288419716939937510582097494459")},
		{"E", X80E, mustParse("2.71828182845904523536028747135266249775724709369995957496696763")},
		{"Ln2", X80Ln2, mustParse("0.693147180559945309417232121458176568075500134360255254120680009")},
		{"Log2E", X80Log2E, mustParse("1.44269504088896340735992468100189213742664595415298593413544941")},
		{"Sqrt2", X80Sqrt2, sqrt2},
	} {
		if want := bigOf(64).Set(c.want); toBig(c.got).Cmp(want) != 0 {
			t.Errorf("X80%s = %s, want %s", c.name, c.got.Internal(), want.Text('g', 25))
		}
	}
}

// relErr returns |got-want|/|want|.
func relErr(got X80, want *big.Float) float64 {
	d := bigOf(300).Sub(toBig(got), want)
	f, _ := d.Quo(d, want).Float64()
	return math.Abs(f)
}

func TestLnAtanAccuracy(t *testing.T) {
	const tol = 0x1p-60 // 16 units in the last place
	ln2 := mustParse("0.693147180559945309417232121458176568075500134360255254120680009")
	pi := mustParse("3.14159265358979323846264338327950288419716939937510582097494459")
	cases := []struct {
		name string
		got  X80
		want *big.Float
	}{
		{"ln(2)", Int64ToFloatX80(2).Ln(), ln2},
		{"ln(10)", Int64ToFloatX80(10).Ln(), mustParse("2.30258509299404568401799145468436420760110148862877297603332790")},
		{"ln(3)", Int64ToFloatX80(3).Ln(), mustParse("1.09861228866810969139524523692252570464749055782274945173469433")},
		{"ln(0.5)", NewFromFloat64(0.5).Ln(), bigOf(300).Neg(ln2)},
		{"ln(e)", X80E.Ln(), mustParse("1.00000000000000000000203678496910777496")},
		{"ln(2^16000)", packFloatX80(false, 0x3FFF+16000, 1<<63).Ln(), bigOf(300).Mul(ln2, bigOf(300).SetInt64(16000))},
		{"ln(2^-16400)", packFloatX80(false, 0, 1<<45).Ln(), bigOf(300).Mul(ln2, bigOf(300).SetInt64(-16400))},
		{"atan(1)", X80One.Atan(), bigOf(300).Quo(pi, bigOf(300).SetInt64(4))},
		{"atan(0.5)", NewFromFloat64(0.5).Atan(), mustParse("0.463647609000806116214256231461214402028537054286120263810933088")},
		{"atan(2)", Int64ToFloatX80(2).Atan(), mustParse("1.10714871779409050301706546017853704007004764540143264667653920")},
		{"atan(-2)", Int64ToFloatX80(-2).Atan(), mustParse("-1.10714871779409050301706546017853704007004764540143264667653920")},
		{"atan(1e-5)", Int64ToFloatX80(1).Div(Int64ToFloatX80(100000)).Atan(), mustParse("0.00000999999999966666666668666666665238095238206349206339")},
		{"atan(+Inf)", X80InfPos.Atan(), bigOf(300).Quo(pi, bigOf(300).SetInt64(2))},
	}
	for _, c := range cases {
		if e := relErr(c.got, c.want); e > tol {
			t.Errorf("%s = %s, relative error %.3g > %.3g", c.name, c.got, e, tol)
		}
	}

	r := rand.New(rand.NewPCG(15, 16))
	for range 20000 {
		x := math.Exp(r.Float64()*1400 - 700)
		if got, want := NewFromFloat64(x).Ln().ToFloat64(), math.Log(x); math.Abs(got-want) > 1e-15*math.Max(1, math.Abs(want)) {
			t.Fatalf("Ln(%v) = %v, want %v", x, got, want)
		}
		y := (r.Float64() - 0.5) * math.Exp(r.Float64()*40-20)
		if got, want := NewFromFloat64(y).Atan().ToFloat64(), math.Atan(y); math.Abs(got-want) > 1e-15*math.Abs(want) {
			t.Fatalf("Atan(%v) = %v, want %v", y, got, want)
		}
	}
}

func TestLnAtanSpecials(t *testing.T) {
	negZero := packFloatX80(true, 0, 0)
	ClearExceptions()
	if got := negZero.Ln(); !got.Eq(X80InfNeg) || !HasException(ExceptionDivbyzero) {
		t.Errorf("Ln(-0) = %s, want -Inf with divide-by-zero", got)
	}
	ClearExceptions()
	if got := X80MinusOne.Ln(); !got.IsNaN() || !HasException(ExceptionInvalid) {
		t.Errorf("Ln(-1) = %s, want NaN with invalid", got)
	}
	if got := X80InfPos.Ln(); !got.Eq(X80InfPos) {
		t.Errorf("Ln(+Inf) = %s, want +Inf", got)
	}
	ClearExceptions()
	if got := X80One.Ln(); !got.Eq(X80Zero) || HasAnyException() {
		t.Errorf("Ln(1) = %s with exceptions %#x, want exact 0", got, GetExceptions())
	}
	if got := negZero.Atan(); got != negZero {
		t.Errorf("Atan(-0) = %s, want -0", got.Internal())
	}

	// intermediate steps must not leak flags or reach the handler
	calls := 0
	SetExceptionHandler(func(int) { calls++ })
	defer SetExceptionHandler(nil)
	ClearExceptions()
	_ = Int64ToFloatX80(3).Ln()
	_ = Int64ToFloatX80(3).Atan()
	if GetExceptions() != ExceptionInexact || calls != 2 {
		t.Errorf("Ln/Atan raised %#x with %d handler calls, want only inexact, once each", GetExceptions(), calls)
	}
}

func TestArithmeticExceptions(t *testing.T) {
	for _, c := range []struct {
		name string
		f    func() X80
		want X80
		exc  int
	}{
		{"1/0", func() X80 { return X80One.Div(X80Zero) }, X80InfPos, ExceptionDivbyzero},
		{"-1/0", func() X80 { return X80MinusOne.Div(X80Zero) }, X80InfNeg, ExceptionDivbyzero},
		{"0/0", func() X80 { return X80Zero.Div(X80Zero) }, X80NaN, ExceptionInvalid},
		{"sqrt(-4)", func() X80 { return Int64ToFloatX80(-4).Sqrt() }, X80NaN, ExceptionInvalid},
		{"sqrt(-0)", func() X80 { return packFloatX80(true, 0, 0).Sqrt() }, packFloatX80(true, 0, 0), 0},
		{"Inf*0", func() X80 { return X80InfPos.Mul(X80Zero) }, X80NaN, ExceptionInvalid},
		{"x rem 0", func() X80 { return X80One.Rem(X80Zero) }, X80NaN, ExceptionInvalid},
	} {
		ClearExceptions()
		got := c.f()
		if got != c.want || GetExceptions() != c.exc {
			t.Errorf("%s = %s with exceptions %#x, want %s with %#x", c.name, got.Internal(), GetExceptions(), c.want.Internal(), c.exc)
		}
	}
}

func TestReducedPrecisionOverflow(t *testing.T) {
	defer func() { RoundingMode, RoundingPrecision = RoundNearestEven, 80 }()
	maxX80 := packFloatX80(false, 0x7FFE, math.MaxUint64)
	for _, c := range []struct {
		prec int
		mask uint64
	}{{64, 0x7FF}, {32, 0xFFFFFFFFFF}} {
		RoundingMode, RoundingPrecision = RoundToZero, c.prec
		want := packFloatX80(false, 0x7FFE, ^c.mask)
		if got := maxX80.Add(maxX80); got != want {
			t.Errorf("precision %d round-to-zero overflow = %s, want %s", c.prec, got.Internal(), want.Internal())
		}
	}
}

func TestBytes(t *testing.T) {
	// x87 memory layout of 1.0: significand, then sign and exponent
	le := []byte{0, 0, 0, 0, 0, 0, 0, 0x80, 0xFF, 0x3F}
	if got := X80One.Bytes(binary.LittleEndian); string(got) != string(le) {
		t.Errorf("Bytes(LittleEndian) = % x, want % x", got, le)
	}
	be := []byte{0x3F, 0xFF, 0x80, 0, 0, 0, 0, 0, 0, 0}
	if got := X80One.Bytes(binary.BigEndian); string(got) != string(be) {
		t.Errorf("Bytes(BigEndian) = % x, want % x", got, be)
	}
	for _, order := range []binary.ByteOrder{binary.LittleEndian, binary.BigEndian} {
		for _, a := range []X80{X80Pi, X80InfNeg, X80NaN, X80Zero, {0x8001, 0x123456789ABCDEF0}} {
			if got := NewFromBytes(a.Bytes(order), order); got != a {
				t.Errorf("NewFromBytes(Bytes(%s)) = %s", a.Internal(), got.Internal())
			}
		}
	}
}

func TestShift128Right(t *testing.T) {
	if z0, z1 := shift128Right(0x8000000000000000, 0, 70); z0 != 0 || z1 != 0x8000000000000000>>6 {
		t.Errorf("shift128Right(2^127, 70) = %x %x", z0, z1)
	}
}
