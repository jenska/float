package float

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"math/rand/v2"
	"testing"
)

var (
	x80Two    = newFromHexString("40008000000000000000") // 2
	x80Half   = newFromHexString("3FFE8000000000000000") // 0.5
	x80HalfPi = newFromHexString("3FFFC90FDAA22168C235") // pi/2
)

var bigModes = map[int]big.RoundingMode{
	RoundNearestEven: big.ToNearestEven,
	RoundToZero:      big.ToZero,
	RoundDown:        big.ToNegativeInf,
	RoundUp:          big.ToPositiveInf,
}

// withEnv runs f with the given rounding mode and precision and restores the
// defaults afterwards.
func withEnv(mode, prec int, f func()) {
	defer func() { RoundingMode, RoundingPrecision = RoundNearestEven, 80 }()
	RoundingMode, RoundingPrecision = mode, prec
	f()
}

func TestUnnormalOperands(t *testing.T) {
	r := rand.New(rand.NewPCG(21, 22))
	for range iterations / 10 {
		a, b := randX80(r, 50), randX80(r, 50)
		// re-encode a as an unnormal with the same value, if no bits are lost
		s := r.IntN(8) + 1
		if a.low&(1<<s-1) != 0 {
			continue
		}
		u := X80{high: a.high + uint16(s), low: a.low >> s}
		for _, c := range []struct {
			name      string
			got, want X80
		}{
			{"Add", u.Add(b), a.Add(b)}, {"Sub", b.Sub(u), b.Sub(a)},
			{"Mul", u.Mul(b), a.Mul(b)}, {"Div", b.Div(u), b.Div(a)},
			{"Rem", u.Rem(b), a.Rem(b)}, {"Sqrt", u.Abs().Sqrt(), a.Abs().Sqrt()},
			{"Normalize", u.Normalize(), a},
		} {
			if c.got != c.want {
				t.Fatalf("%s with unnormal %s = %s, want %s", c.name, u.Internal(), c.got.Internal(), c.want.Internal())
			}
		}
		if !u.Eq(a) || u.Lt(a) || u.ToFloat64() != a.ToFloat64() || u.String() != a.String() {
			t.Fatalf("unnormal %s does not behave like %s", u.Internal(), a.Internal())
		}
	}

	half := X80{0x3FFF, 0x4000000000000000} // 0.5 as an unnormal
	if got := half.Add(X80Zero); got != x80Half {
		t.Errorf("unnormal 0.5 + 0 = %s, want 0.5", got)
	}
	pseudoDenormal := X80{0, 1 << 63} // the smallest normal, encoded with exponent 0
	if want := (X80{1, 1 << 63}); pseudoDenormal.Normalize() != want || !pseudoDenormal.Eq(want) {
		t.Errorf("pseudo-denormal %s not equal to %s", pseudoDenormal.Internal(), want.Internal())
	}
	pseudoInf := X80{0x7FFF, 0}
	if !pseudoInf.IsInf() || pseudoInf.Add(X80One) != X80InfPos {
		t.Errorf("pseudo-infinity %s is not treated as +Inf", pseudoInf.Internal())
	}
	if (X80{0x4000, 0}).Normalize() != X80Zero {
		t.Error("unnormal zero does not normalize to zero")
	}
	if tiny := (X80{2, 1 << 61}); tiny.Normalize() != (X80{0, 1 << 62}) {
		t.Errorf("unnormal below the normal range normalizes to %s", tiny.Normalize().Internal())
	}
}

func TestDefaultNaN(t *testing.T) {
	defer func(saved X80) { DefaultNaN = saved }(DefaultNaN)
	DefaultNaN = NewFromBits(0x7FFF, math.MaxUint64)
	for name, got := range map[string]X80{
		"0/0": X80Zero.Div(X80Zero), "Inf-Inf": X80InfPos.Sub(X80InfPos),
		"sqrt(-1)": X80MinusOne.Sqrt(), "ln(-1)": X80MinusOne.Ln(), "sin(Inf)": X80InfPos.Sin(),
	} {
		if got != DefaultNaN {
			t.Errorf("%s = %s, want DefaultNaN", name, got.Internal())
		}
	}
	// NaN operands propagate instead
	if got := X80NaN.Add(X80One); got != X80NaN {
		t.Errorf("NaN + 1 = %s, want the operand NaN", got.Internal())
	}
}

func TestBitsAndBytes(t *testing.T) {
	a := NewFromBits(0xC123, 0x8765432112345678)
	if hi, lo := a.Bits(); hi != 0xC123 || lo != 0x8765432112345678 {
		t.Errorf("Bits() = %04X %016X", hi, lo)
	}
	be := []byte{0x3F, 0xFF, 0, 0, 0x80, 0, 0, 0, 0, 0, 0, 0}
	if got := X80One.Bytes96(binary.BigEndian); string(got) != string(be) {
		t.Errorf("Bytes96(BigEndian) = % x, want % x (68881 layout)", got, be)
	}
	le := []byte{0, 0, 0, 0, 0, 0, 0, 0x80, 0xFF, 0x3F, 0, 0}
	if got := X80One.Bytes96(binary.LittleEndian); string(got) != string(le) {
		t.Errorf("Bytes96(LittleEndian) = % x, want % x (x87 layout)", got, le)
	}
	for _, order := range []binary.ByteOrder{binary.BigEndian, binary.LittleEndian} {
		if got := NewFromBytes96(a.Bytes96(order), order); got != a {
			t.Errorf("NewFromBytes96(Bytes96(%s)) = %s", a.Internal(), got.Internal())
		}
	}
}

func TestClassification(t *testing.T) {
	nan := NewFromBits(0xFFFF, 0x8000000000000001) // signaling NaN with sign bit
	if nan.Abs() != NewFromBits(0x7FFF, 0x8000000000000001) || nan.Neg() != NewFromBits(0x7FFF, 0x8000000000000001) {
		t.Error("Abs/Neg must only change the sign bit of a NaN")
	}
	if !nan.Signbit() || !packFloatX80(true, 0, 0).Signbit() || X80Zero.Signbit() {
		t.Error("Signbit")
	}
	if !packFloatX80(true, 0, 0).IsZero() || X80One.IsZero() || !(X80{0x4000, 0}).IsZero() {
		t.Error("IsZero")
	}
	if !(X80{0, 1}).IsSubnormal() || X80Zero.IsSubnormal() || (X80{0, 1 << 63}).IsSubnormal() {
		t.Error("IsSubnormal")
	}
}

func TestFloatBits(t *testing.T) {
	r := rand.New(rand.NewPCG(23, 24))
	for range iterations {
		b64 := r.Uint64()
		if f := math.Float64frombits(b64); !math.IsNaN(f) {
			if got := NewFromFloat64Bits(b64).ToFloat64Bits(); got != b64 {
				t.Fatalf("float64 %016X round-trips to %016X", b64, got)
			}
		}
		b32 := r.Uint32()
		if f := math.Float32frombits(b32); !math.IsNaN(float64(f)) {
			if got := NewFromFloat32Bits(b32).ToFloat32Bits(); got != b32 {
				t.Fatalf("float32 %08X round-trips to %08X", b32, got)
			}
		}
	}
	for _, c := range []struct {
		name      string
		in        func() X80
		want      X80
		invalid   bool
		back      func(X80) uint64
		wantBack  uint64
		backValid bool
	}{
		{"float32 sNaN", func() X80 { return NewFromFloat32Bits(0xFF800001) }, NewFromBits(0xFFFF, 0xC000010000000000), true,
			func(x X80) uint64 { return uint64(x.ToFloat32Bits()) }, 0xFFC00001, false},
		{"float32 qNaN", func() X80 { return NewFromFloat32Bits(0x7FC12345) }, NewFromBits(0x7FFF, 0xC123450000000000), false,
			func(x X80) uint64 { return uint64(x.ToFloat32Bits()) }, 0x7FC12345, false},
		{"float64 sNaN", func() X80 { return NewFromFloat64Bits(0x7FF0000000000001) }, NewFromBits(0x7FFF, 0xC000000000000800), true,
			func(x X80) uint64 { return x.ToFloat64Bits() }, 0x7FF8000000000001, false},
	} {
		ClearExceptions()
		got := c.in()
		if got != c.want || HasException(ExceptionInvalid) != c.invalid {
			t.Errorf("%s = %s invalid=%v, want %s invalid=%v", c.name, got.Internal(), HasException(ExceptionInvalid), c.want.Internal(), c.invalid)
		}
		if back := c.back(got); back != c.wantBack {
			t.Errorf("%s converts back to %X, want %X", c.name, back, c.wantBack)
		}
	}
	ClearExceptions()
	NewFromBits(0x7FFF, 0x8000000000000001).ToFloat64Bits()
	if !HasException(ExceptionInvalid) {
		t.Error("converting a signaling NaN to float64 must raise invalid")
	}
}

func TestToInt16And8(t *testing.T) {
	for _, c := range []struct {
		a     X80
		mode  int
		want  int64
		exc   int
		int16 bool
	}{
		{Int64ToFloatX80(32767), RoundNearestEven, 32767, 0, true},
		{Int64ToFloatX80(-32768), RoundNearestEven, -32768, 0, true},
		{Int64ToFloatX80(32768), RoundNearestEven, 32767, ExceptionInvalid, true},
		{Int64ToFloatX80(-40000), RoundNearestEven, -32768, ExceptionInvalid, true},
		{X80NaN, RoundNearestEven, 32767, ExceptionInvalid, true},
		{NewFromFloat64(2.5), RoundNearestEven, 2, ExceptionInexact, true},
		{NewFromFloat64(-2.5), RoundDown, -3, ExceptionInexact, false},
		{NewFromFloat64(127.9), RoundToZero, 127, ExceptionInexact, false},
		{NewFromFloat64(127.9), RoundNearestEven, 127, ExceptionInvalid, false},
		{NewFromFloat64(-128.4), RoundNearestEven, -128, ExceptionInexact, false},
		{X80InfNeg, RoundNearestEven, -128, ExceptionInvalid, false},
	} {
		withEnv(c.mode, 80, func() {
			ClearExceptions()
			var got int64
			if c.int16 {
				got = int64(c.a.ToInt16())
			} else {
				got = int64(c.a.ToInt8())
			}
			if got != c.want || GetExceptions() != c.exc {
				t.Errorf("ToInt%d(%s) mode %d = %d exc %#x, want %d exc %#x", map[bool]int{true: 16, false: 8}[c.int16], c.a, c.mode, got, GetExceptions(), c.want, c.exc)
			}
		})
	}
}

// remQuoRef computes the remainder and quotient of a/b exactly, rounding the
// quotient to nearest-even or toward zero.
func remQuoRef(a, b X80, truncate bool) (*big.Float, *big.Int) {
	ab, bb := toBig(a), toBig(b)
	am, bm := new(big.Int), new(big.Int)
	ae, be := ab.MantExp(nil)-64, bb.MantExp(nil)-64
	new(big.Float).SetMantExp(ab, -ae).Int(am)
	new(big.Float).SetMantExp(bb, -be).Int(bm)
	e := min(ae, be)
	am.Lsh(am, uint(ae-e))
	bm.Lsh(bm, uint(be-e))
	n, rem := new(big.Int).QuoRem(am, bm, new(big.Int)) // truncated
	if !truncate {
		twice := new(big.Int).Abs(new(big.Int).Lsh(rem, 1))
		if c := twice.CmpAbs(bm); c > 0 || (c == 0 && n.Bit(0) == 1) {
			if rem.Sign() == bm.Sign() {
				rem.Sub(rem, bm)
				n.Add(n, big.NewInt(1))
			} else {
				rem.Add(rem, bm)
				n.Sub(n, big.NewInt(1))
			}
		}
	}
	z := new(big.Float).SetInt(rem)
	return z.SetMantExp(z, e), n
}

func TestModRemQuo(t *testing.T) {
	r := rand.New(rand.NewPCG(25, 26))
	for i := range iterations {
		a, b := randX80(r, 300), randX80(r, 20)
		if i%3 == 0 {
			a = Int64ToFloatX80(r.Int64N(1<<40) - 1<<39)
			b = Int64ToFloatX80(r.Int64N(1000) + 1)
		}
		for _, truncate := range []bool{false, true} {
			var got X80
			var quo int
			if truncate {
				got, quo = a.ModQuo(b)
			} else {
				got, quo = a.RemQuo(b)
			}
			want, n := remQuoRef(a, b, truncate)
			wantQuo := int(new(big.Int).And(new(big.Int).Abs(n), big.NewInt(0x7FFFFFFF)).Int64())
			if n.Sign() < 0 || (n.Sign() == 0 && a.sign() != b.sign()) {
				wantQuo = -wantQuo
			}
			if toBig(got).Cmp(want) != 0 || quo != wantQuo {
				t.Fatalf("truncate=%v: %s rem %s = %s quo %d, want %s quo %d", truncate, a.Internal(), b.Internal(), got.Internal(), quo, want.Text('g', 20), wantQuo)
			}
		}
	}
	if got, quo := NewFromFloat64(-7).ModQuo(NewFromFloat64(2)); got != NewFromFloat64(-1) || quo != -3 {
		t.Errorf("-7 mod 2 = %s quo %d, want -1 quo -3", got, quo)
	}
	if got, quo := NewFromFloat64(7).RemQuo(NewFromFloat64(2)); got != NewFromFloat64(-1) || quo != 4 {
		t.Errorf("7 rem 2 = %s quo %d, want -1 quo 4", got, quo)
	}
}

func TestScaleGetExpGetMan(t *testing.T) {
	r := rand.New(rand.NewPCG(27, 28))
	for range iterations {
		a := randX80(r, 16000)
		n := r.IntN(80000) - 40000
		want := new(big.Float).SetMantExp(toBig(a), n)
		if inNormalRange(want) {
			if got := a.Scale(n); toBig(got).Cmp(want) != 0 {
				t.Fatalf("%s.Scale(%d) = %s", a.Internal(), n, got.Internal())
			}
		}
		e, m := a.GetExp(), a.GetMan()
		if got := m.Scale(int(e.ToInt64())); got != a || m.Abs().Lt(X80One) || m.Abs().Ge(x80Two) {
			t.Fatalf("GetMan(%s) = %s, GetExp = %s", a.Internal(), m, e)
		}
	}
	check := func(name string, f func() X80, want X80, exc int) {
		t.Helper()
		ClearExceptions()
		if got := f(); got != want || GetExceptions() != exc {
			t.Errorf("%s = %s exc %#x, want %s exc %#x", name, got.Internal(), GetExceptions(), want.Internal(), exc)
		}
	}
	check("1*2^-16445", func() X80 { return X80One.Scale(-16445) }, X80{0, 1}, 0)
	check("1*2^-16446", func() X80 { return X80One.Scale(-16446) }, X80Zero, ExceptionUnderflow|ExceptionInexact)
	check("1.5*2^-16446", func() X80 { return NewFromFloat64(1.5).Scale(-16446) }, X80{0, 1}, ExceptionUnderflow|ExceptionInexact)
	check("1*2^16384", func() X80 { return X80One.Scale(16384) }, X80InfPos, ExceptionOverflow|ExceptionInexact)
	check("1*2^maxint", func() X80 { return X80One.Scale(math.MaxInt) }, X80InfPos, ExceptionOverflow|ExceptionInexact)
	check("max*2^-maxint", func() X80 { return NewFromBits(0x7FFE, math.MaxUint64).Scale(math.MinInt) }, X80Zero, ExceptionUnderflow|ExceptionInexact)
	check("Inf*2^-5", func() X80 { return X80InfNeg.Scale(-5) }, X80InfNeg, 0)
	check("GetExp(2^-16445)", func() X80 { return X80{0, 1}.GetExp() }, Int64ToFloatX80(-16445), 0)
	check("GetMan(2^-16445)", func() X80 { return X80{0, 1}.GetMan() }, X80One, 0)
	check("GetExp(-0)", func() X80 { return packFloatX80(true, 0, 0).GetExp() }, packFloatX80(true, 0, 0), 0)
	check("GetExp(Inf)", func() X80 { return X80InfPos.GetExp() }, DefaultNaN, ExceptionInvalid)
	check("GetMan(Inf)", func() X80 { return X80InfPos.GetMan() }, DefaultNaN, ExceptionInvalid)
	withEnv(RoundToZero, 80, func() {
		check("RZ 1*2^16384", func() X80 { return X80One.Scale(16384) }, NewFromBits(0x7FFE, math.MaxUint64), ExceptionOverflow|ExceptionInexact)
	})
}

// truncate24 clears all but the top 24 significand bits.
func truncate24(a X80) X80 {
	a.low &= 0xFFFFFF0000000000
	return a
}

func TestSglMulSglDivRoundToPrecision(t *testing.T) {
	r := rand.New(rand.NewPCG(29, 30))
	for range iterations {
		a, b := randX80(r, 200), randX80(r, 200)
		mode := r.IntN(4)
		withEnv(mode, 64, func() { // RoundingPrecision must not matter
			bm := bigModes[mode]
			want := new(big.Float).SetPrec(24).SetMode(bm).Mul(toBig(truncate24(a)), toBig(truncate24(b)))
			if got := a.SglMul(b); toBig(got).Cmp(want) != 0 {
				t.Fatalf("mode %d: %s SglMul %s = %s, want %s", mode, a.Internal(), b.Internal(), got.Internal(), want.Text('p', 0))
			}
			want = new(big.Float).SetPrec(24).SetMode(bm).Quo(toBig(a), toBig(b))
			if got := a.SglDiv(b); toBig(got).Cmp(want) != 0 {
				t.Fatalf("mode %d: %s SglDiv %s = %s, want %s", mode, a.Internal(), b.Internal(), got.Internal(), want.Text('p', 0))
			}
			for prec, bits := range map[int]uint{32: 24, 64: 53, 80: 64} {
				want := new(big.Float).SetPrec(bits).SetMode(bm).Set(toBig(a))
				if got := a.RoundToPrecision(prec); toBig(got).Cmp(want) != 0 {
					t.Fatalf("mode %d: %s.RoundToPrecision(%d) = %s, want %s", mode, a.Internal(), prec, got.Internal(), want.Text('p', 0))
				}
			}
			want = new(big.Float).SetMode(big.ToZero)
			n, _ := toBig(a).Int(nil)
			want.SetInt(n)
			if got := a.Trunc(); toBig(got).Cmp(want) != 0 && !(n.Sign() == 0) {
				t.Fatalf("Trunc(%s) = %s, want %s", a.Internal(), got.Internal(), want.Text('g', 25))
			}
			if RoundingMode != mode {
				t.Fatal("Trunc did not restore the rounding mode")
			}
		})
	}
}

func TestConstantValue(t *testing.T) {
	one := rInt(1)
	refs := map[Constant]*big.Float{
		ConstPi: refPi(refPrec), ConstE: refExp(one), ConstLn2: refLn2, ConstLn10: refLn10,
		ConstLog2E: rf().Quo(one, refLn2), ConstLog10E: rf().Quo(one, refLn10), ConstLog10Of2: rf().Quo(refLn2, refLn10),
	}
	nearest := map[Constant]X80{
		ConstPi: X80Pi, ConstE: X80E, ConstLn2: X80Ln2, ConstLn10: X80Ln10,
		ConstLog2E: X80Log2E, ConstLog10E: X80Log10E, ConstLog10Of2: X80Log10Of2,
	}
	for c, ref := range refs {
		if got := c.Value(); got != nearest[c] {
			t.Errorf("Constant %d: Value() = %s, X80 variable = %s", c, got.Internal(), nearest[c].Internal())
		}
		for mode, bm := range bigModes {
			for prec, bits := range map[int]uint{32: 24, 64: 53, 80: 64} {
				withEnv(mode, prec, func() {
					ClearExceptions()
					want := new(big.Float).SetPrec(bits).SetMode(bm).Set(ref)
					if got := c.Value(); toBig(got).Cmp(want) != 0 || !HasException(ExceptionInexact) {
						t.Errorf("Constant %d mode %d prec %d: Value() = %s, want %s", c, mode, prec, got.Internal(), want.Text('p', 0))
					}
				})
			}
		}
	}
}

func TestPow10AndParse(t *testing.T) {
	ten := big.NewInt(10)
	exact := func(n int) *big.Float {
		p := new(big.Int).Exp(ten, big.NewInt(int64(max(n, -n))), nil)
		if n >= 0 {
			return new(big.Float).SetInt(p)
		}
		return new(big.Float).SetPrec(20000).Quo(big.NewFloat(1).SetPrec(20000), new(big.Float).SetPrec(20000).SetInt(p))
	}
	r := rand.New(rand.NewPCG(31, 32))
	for i := range 2000 {
		n := r.IntN(9860) - 4930
		if i < 60 {
			n = i - 30
		}
		mode := RoundNearestEven
		if i%2 == 1 {
			mode = r.IntN(4)
		}
		withEnv(mode, 80, func() {
			ClearExceptions()
			want := new(big.Float).SetPrec(64).SetMode(bigModes[mode]).Set(exact(n))
			got := Pow10(n)
			if inNormalRange(want) && toBig(got).Cmp(want) != 0 {
				t.Fatalf("mode %d: Pow10(%d) = %s, want %s", mode, n, got.Internal(), want.Text('g', 25))
			}
			if inexact := HasException(ExceptionInexact); inexact != (n < 0 || n > 27) {
				t.Fatalf("Pow10(%d) inexact = %v", n, inexact)
			}
		})
	}

	for range 5000 {
		digits := make([]byte, r.IntN(40)+1)
		for j := range digits {
			digits[j] = byte('0' + r.IntN(10))
		}
		exp := r.IntN(9800) - 4900
		s := fmt.Sprintf("%s.%se%d", digits[:1], digits[1:], exp)
		if r.IntN(2) == 0 {
			s = "-" + s
		}
		checkParse(t, s, r.IntN(4), []int{32, 64, 80}[r.IntN(3)])
	}
	// midpoints and representable values right at the rounding boundaries of
	// the fast path, which must fall back to the exact conversion
	for _, s := range []string{
		"18446744073709551617", "18446744073709551619", "18446744073709551615", "36893488147419103233",
		"9007199254740993", "9007199254740995", "16777217", "33554435", "1.8446744073709551617e5",
		"340282366920938463463374607431768211457", "0.0000000000000000000542101086242752217003726400434970855712890625",
	} {
		for mode := range 4 {
			for _, prec := range []int{32, 64, 80} {
				checkParse(t, s, mode, prec)
				checkParse(t, "-"+s, mode, prec)
			}
		}
	}
	for s, want := range map[string]X80{
		"0x1.8p3": Int64ToFloatX80(12), "-0X.8P-1": NewFromFloat64(-0.25), "1": X80One, "+.5": x80Half,
		"1E2": Int64ToFloatX80(100), "-0": packFloatX80(true, 0, 0), "INF": X80InfPos, "-infinity": X80InfNeg,
		"3.14159265358979323846264338327950288": X80Pi, "0x1p-16445": {0, 1},
	} {
		if got, err := Parse(s); got != want || err != nil {
			t.Errorf("Parse(%q) = %s, %v; want %s", s, got.Internal(), err, want.Internal())
		}
	}
	if got, _ := Parse("nan"); !got.IsNaN() {
		t.Error(`Parse("nan") is not a NaN`)
	}
	for _, s := range []string{"", "-", ".", "e5", "1e", "1e+", "1.2.3", "0x", "0x1p", "1f", "--1", "1e5x"} {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) succeeded", s)
		}
	}
}

func TestTranscendentalSpecials(t *testing.T) {
	negZero := packFloatX80(true, 0, 0)
	const none, inex, inv, dz = 0, ExceptionInexact, ExceptionInvalid, ExceptionDivbyzero
	ovf, unf := ExceptionOverflow|ExceptionInexact, ExceptionUnderflow|ExceptionInexact
	for _, c := range []struct {
		name string
		f    func() X80
		want X80
		exc  int
	}{
		{"Exp(0)", X80Zero.Exp, X80One, none},
		{"Exp(-Inf)", X80InfNeg.Exp, X80Zero, none},
		{"Exp(+Inf)", X80InfPos.Exp, X80InfPos, none},
		{"Exp(20000)", Int64ToFloatX80(20000).Exp, X80InfPos, ovf},
		{"Exp(-20000)", Int64ToFloatX80(-20000).Exp, X80Zero, unf},
		{"Exp2(10)", Int64ToFloatX80(10).Exp2, Int64ToFloatX80(1024), none},
		{"Exp2(-16445)", Int64ToFloatX80(-16445).Exp2, X80{0, 1}, none},
		{"Exp10(3)", Int64ToFloatX80(3).Exp10, Int64ToFloatX80(1000), none},
		{"Expm1(-0)", negZero.Expm1, negZero, none},
		{"Expm1(-Inf)", X80InfNeg.Expm1, X80MinusOne, none},
		{"Ln(-0)", negZero.Ln, X80InfNeg, dz},
		{"Ln(1)", X80One.Ln, X80Zero, none},
		{"Log2(2^-16445)", X80{0, 1}.Log2, Int64ToFloatX80(-16445), none},
		{"Log2(1024)", Int64ToFloatX80(1024).Log2, Int64ToFloatX80(10), none},
		{"Log10(1e27)", Pow10(27).Log10, Int64ToFloatX80(27), none},
		{"Log10(-1)", X80MinusOne.Log10, DefaultNaN, inv},
		{"Log1p(-1)", X80MinusOne.Log1p, X80InfNeg, dz},
		{"Log1p(-2)", Int64ToFloatX80(-2).Log1p, DefaultNaN, inv},
		{"Log1p(-0)", negZero.Log1p, negZero, none},
		{"Sin(-0)", negZero.Sin, negZero, none},
		{"Sin(Inf)", X80InfPos.Sin, DefaultNaN, inv},
		{"Cos(-0)", negZero.Cos, X80One, none},
		{"Tan(-0)", negZero.Tan, negZero, none},
		{"Tan(-Inf)", X80InfNeg.Tan, DefaultNaN, inv},
		{"Atan(-Inf)", X80InfNeg.Atan, x80HalfPi.Neg(), inex},
		{"Asin(-1)", X80MinusOne.Asin, x80HalfPi.Neg(), inex},
		{"Asin(2)", x80Two.Asin, DefaultNaN, inv},
		{"Acos(1)", X80One.Acos, X80Zero, none},
		{"Acos(-1)", X80MinusOne.Acos, X80Pi, inex},
		{"Acos(-Inf)", X80InfNeg.Acos, DefaultNaN, inv},
		{"Sinh(-Inf)", X80InfNeg.Sinh, X80InfNeg, none},
		{"Sinh(-12000)", Int64ToFloatX80(-12000).Sinh, X80InfNeg, ovf},
		{"Cosh(-Inf)", X80InfNeg.Cosh, X80InfPos, none},
		{"Cosh(0)", negZero.Cosh, X80One, none},
		{"Tanh(-Inf)", X80InfNeg.Tanh, X80MinusOne, none},
		{"Tanh(100)", Int64ToFloatX80(100).Tanh, X80One, inex},
		{"Atanh(-1)", X80MinusOne.Atanh, X80InfNeg, dz},
		{"Atanh(Inf)", X80InfPos.Atanh, DefaultNaN, inv},
		{"Atanh(-0)", negZero.Atanh, negZero, none},
		{"Sin(subnormal)", X80{0, 5}.Sin, X80{0, 5}, unf},
	} {
		ClearExceptions()
		if got := c.f(); got != c.want || GetExceptions() != c.exc {
			t.Errorf("%s = %s exc %#x, want %s exc %#x", c.name, got.Internal(), GetExceptions(), c.want.Internal(), c.exc)
		}
	}

	// NaNs propagate through every function; signaling NaNs raise invalid
	sNaN := NewFromBits(0x7FFF, 0x8000000000000123)
	for _, tf := range transcendentals {
		ClearExceptions()
		if got := tf.f(sNaN); got != NewFromBits(0x7FFF, 0xC000000000000123) || GetExceptions() != ExceptionInvalid {
			t.Errorf("%s(sNaN) = %s exc %#x, want the quieted NaN with invalid", tf.name, got.Internal(), GetExceptions())
		}
	}
}

func TestTranscendentalEnvironment(t *testing.T) {
	// intermediate operations must neither leak flags nor reach the handler
	calls := 0
	SetExceptionHandler(func(int) { calls++ })
	defer SetExceptionHandler(nil)
	x := NewFromFloat64(0.3) // inside every domain, no exact result
	for _, tf := range transcendentals {
		ClearExceptions()
		calls = 0
		tf.f(x)
		if GetExceptions() != ExceptionInexact || calls != 1 {
			t.Errorf("%s(%s) raised %#x with %d handler calls, want inexact once", tf.name, x, GetExceptions(), calls)
		}
	}
	ClearExceptions()
	calls = 0
	s, c := x.Sincos()
	if s != x.Sin() || c != x.Cos() {
		t.Error("Sincos differs from Sin and Cos")
	}
	if calls != 3 {
		t.Errorf("Sincos called the handler %d times, want once", calls-2)
	}

	// results honor the rounding precision
	withEnv(RoundNearestEven, 32, func() {
		for _, tf := range transcendentals {
			x := x80Half
			if got := tf.f(x); got.low&0xFFFFFFFFFF != 0 {
				t.Errorf("%s with precision 32 = %s, not rounded to 24 bits", tf.name, got.Internal())
			}
		}
	})
}

// checkParse compares Parse(s) under the given rounding mode and precision
// with the exact conversion by math/big.
func checkParse(t *testing.T, s string, mode, prec int) {
	t.Helper()
	rat, ok := new(big.Rat).SetString(s)
	if !ok {
		t.Fatalf("bad test input %q", s)
	}
	bits := map[int]uint{32: 24, 64: 53, 80: 64}[prec]
	want := new(big.Float).SetPrec(bits).SetMode(bigModes[mode]).SetRat(rat)
	withEnv(mode, prec, func() {
		ClearExceptions()
		got, err := Parse(s)
		if err != nil {
			t.Fatalf("Parse(%q): %v", s, err)
		}
		if !inNormalRange(want) {
			return
		}
		exact := new(big.Rat).SetFrac(big.NewInt(0), big.NewInt(1))
		want.Rat(exact)
		if toBig(got).Cmp(want) != 0 || HasException(ExceptionInexact) != (exact.Cmp(rat) != 0) {
			t.Fatalf("mode %d prec %d: Parse(%q) = %s inexact=%v, want %s", mode, prec, s, got.Internal(), HasException(ExceptionInexact), want.Text('g', 30))
		}
	})
}
