package float

// Accuracy tests of the transcendental functions against 400-bit references
// computed with math/big.

import (
	"math"
	"math/big"
	"math/rand/v2"
	"sync"
	"testing"
)

const refPrec = 400

func rf() *big.Float { return new(big.Float).SetPrec(refPrec) }

func rInt(n int64) *big.Float { return rf().SetInt64(n) }

// refAtanhInv returns atanh(1/n).
func refAtanhInv(n int64) *big.Float {
	sum, p := rf(), rf().Quo(rInt(1), rInt(n))
	n2 := rInt(n * n)
	for k := int64(0); p.MantExp(nil) > -refPrec-10; k++ {
		sum.Add(sum, rf().Quo(p, rInt(2*k+1)))
		p.Quo(p, n2)
	}
	return sum
}

var refLn2 = rf().Mul(rInt(2), refAtanhInv(3))
var refLn10 = rf().Add(rf().Mul(rInt(3), refLn2), rf().Mul(rInt(2), refAtanhInv(9)))

// refPi returns pi with prec bits (Machin's formula).
func refPi(prec uint) *big.Float {
	atanInv := func(n int64) *big.Float {
		sum := new(big.Float).SetPrec(prec)
		p := new(big.Float).SetPrec(prec).Quo(big.NewFloat(1), big.NewFloat(float64(n)))
		n2 := new(big.Float).SetPrec(prec).SetInt64(n * n)
		for k := int64(0); p.Sign() != 0 && p.MantExp(nil) > -int(prec)-10; k++ {
			t := new(big.Float).SetPrec(prec).Quo(p, new(big.Float).SetInt64(2*k+1))
			if k&1 != 0 {
				t.Neg(t)
			}
			sum.Add(sum, t)
			p.Quo(p, n2)
		}
		return sum
	}
	pi := new(big.Float).SetPrec(prec).Mul(big.NewFloat(16), atanInv(5))
	return pi.Sub(pi, new(big.Float).SetPrec(prec).Mul(big.NewFloat(4), atanInv(239)))
}

var refPiBig = sync.OnceValue(func() *big.Float { return refPi(16384 + 2*refPrec) })

func refExp(x *big.Float) *big.Float {
	if x.Sign() == 0 {
		return rInt(1)
	}
	// e^x = (e^(x/2^k))^(2^k) with |x/2^k| < 2^-20
	k := max(x.MantExp(nil)+20, 0)
	y := rf().SetMantExp(x, -k)
	sum, term := rInt(1), rInt(1)
	for n := int64(1); term.Sign() != 0 && term.MantExp(nil) > -refPrec-10; n++ {
		term.Mul(term, y).Quo(term, rInt(n))
		sum.Add(sum, term)
	}
	for range k {
		sum.Mul(sum, sum)
	}
	return sum
}

func refLn(x *big.Float) *big.Float {
	m := rf()
	e := x.MantExp(m) // x = m*2^e, m in [0.5, 1)
	s := rf().Quo(rf().Sub(m, rInt(1)), rf().Add(m, rInt(1)))
	s2 := rf().Mul(s, s)
	sum, p := rf(), rf().Set(s)
	for k := int64(0); p.Sign() != 0 && p.MantExp(nil) > -refPrec-10; k++ {
		sum.Add(sum, rf().Quo(p, rInt(2*k+1)))
		p.Mul(p, s2)
	}
	sum.Mul(sum, rInt(2))
	return sum.Add(sum, rf().Mul(rInt(int64(e)), refLn2))
}

func refAtan(x *big.Float) *big.Float {
	if x.Sign() < 0 {
		return rf().Neg(refAtan(rf().Neg(x)))
	}
	halfPi := rf().Quo(refPi(refPrec), rInt(2))
	if x.Cmp(rInt(1)) > 0 {
		return rf().Sub(halfPi, refAtan(rf().Quo(rInt(1), x)))
	}
	y := rf().Set(x)
	for range 4 { // atan(x) = 2*atan(x/(1+sqrt(1+x^2)))
		y.Quo(y, rf().Add(rInt(1), rf().Sqrt(rf().Add(rInt(1), rf().Mul(y, y)))))
	}
	y2 := rf().Mul(y, y)
	sum, p := rf(), rf().Set(y)
	for k := int64(0); p.Sign() != 0 && p.MantExp(nil) > -refPrec-10; k++ {
		t := rf().Quo(p, rInt(2*k+1))
		if k&1 != 0 {
			t.Neg(t)
		}
		sum.Add(sum, t)
		p.Mul(p, y2)
	}
	return sum.Mul(sum, rInt(16))
}

// refSinCos returns sin(x) and cos(x), reducing x with enough bits of pi for
// any X80 argument.
func refSinCos(x *big.Float) (sin, cos *big.Float) {
	prec := uint(max(x.MantExp(nil), 0)) + 2*refPrec
	pi := refPiBig()
	halfPi := new(big.Float).SetPrec(prec).Quo(pi, big.NewFloat(2))
	q := new(big.Float).SetPrec(prec).Quo(x, halfPi)
	nInt, _ := new(big.Float).SetPrec(prec).Add(q, big.NewFloat(0.5).SetPrec(prec)).Int(nil)
	if q.Sign() < 0 { // round to nearest for negative quotients
		nInt, _ = new(big.Float).SetPrec(prec).Sub(q, big.NewFloat(0.5)).Int(nil)
	}
	r := new(big.Float).SetPrec(prec).Sub(x, new(big.Float).SetPrec(prec).Mul(new(big.Float).SetInt(nInt), halfPi))
	r = rf().Set(r)
	n := new(big.Int).Mod(nInt, big.NewInt(4)).Int64()
	r2 := rf().Mul(r, r)
	s, c := rf().Set(r), rInt(1)
	ts, tc := rf().Set(r), rInt(1)
	for k := int64(1); ts.Sign() != 0 && ts.MantExp(nil) > -refPrec-10; k++ {
		ts.Mul(ts, r2).Quo(ts, rInt(-(2*k)*(2*k+1)))
		s.Add(s, ts)
		tc.Mul(tc, r2).Quo(tc, rInt(-(2*k-1)*(2*k)))
		c.Add(c, tc)
	}
	switch n {
	case 1:
		return c, s.Neg(s)
	case 2:
		return s.Neg(s), c.Neg(c)
	case 3:
		return c.Neg(c), s
	}
	return s, c
}

// ulpErr returns |got - want| in units of the last place of want.
func ulpErr(got X80, want *big.Float) float64 {
	if want.Sign() == 0 {
		if got.IsZero() {
			return 0
		}
		return math.Inf(1)
	}
	e := want.MantExp(nil) - 1 // want = 1.f * 2^e
	ulp := rf().SetMantExp(rInt(1), max(e, -0x3FFE)-63)
	d := rf().Sub(toBig(got), want)
	f, _ := d.Quo(d.Abs(d), ulp).Float64()
	return f
}

type transcendental struct {
	name string
	f    func(X80) X80
	ref  func(*big.Float) *big.Float
	gen  func(r *rand.Rand) X80
	tol  float64 // maximum error in ulps
}

// logUniform returns a value with random sign and magnitude 2^[lo, hi),
// subnormal where the exponent is below the normal range.
func logUniform(r *rand.Rand, lo, hi int, positive bool) X80 {
	sign, exp, sig := !positive && r.IntN(2) == 0, 0x3FFF+lo+r.IntN(hi-lo), r.Uint64()|1<<63
	if exp < 1 {
		return packFloatX80(sign, 0, sig>>min(1-exp, 63))
	}
	return packFloatX80(sign, exp, sig)
}

// below returns a random value in (-limit, limit), or [0, limit) if positive.
func below(r *rand.Rand, limit float64, positive bool) X80 {
	x := NewFromFloat64(r.Float64() * limit)
	if x.low != 0 {
		x.low |= r.Uint64() & 0x7FF // fill the bits below float64 precision
	}
	if !positive && r.IntN(2) == 0 {
		x = x.Neg()
	}
	return x
}

func pick(r *rand.Rand, gens ...func() X80) X80 { return gens[r.IntN(len(gens))]() }

var transcendentals = []transcendental{
	{"Exp", X80.Exp, refExp, func(r *rand.Rand) X80 {
		return pick(r, func() X80 { return below(r, 11355, false) }, func() X80 { return logUniform(r, -70, 4, false) })
	}, 0.501},
	{"Exp2", X80.Exp2, func(x *big.Float) *big.Float { return refExp(rf().Mul(x, refLn2)) }, func(r *rand.Rand) X80 {
		return pick(r, func() X80 { return below(r, 16383, false) }, func() X80 { return logUniform(r, -70, 4, false) })
	}, 0.501},
	{"Exp10", X80.Exp10, func(x *big.Float) *big.Float { return refExp(rf().Mul(x, refLn10)) }, func(r *rand.Rand) X80 {
		return pick(r, func() X80 { return below(r, 4931, false) }, func() X80 { return logUniform(r, -70, 4, false) })
	}, 0.501},
	{"Expm1", X80.Expm1, func(x *big.Float) *big.Float { return rf().Sub(refExp(x), rInt(1)) }, func(r *rand.Rand) X80 {
		return pick(r, func() X80 { return below(r, 70, false) }, func() X80 { return logUniform(r, -70, 2, false) })
	}, 0.501},
	{"Ln", X80.Ln, refLn, func(r *rand.Rand) X80 {
		return pick(r, func() X80 { return logUniform(r, -16440, 16383, true) },
			func() X80 { return X80One.Add(logUniform(r, -70, -1, false)) })
	}, 0.501},
	{"Log2", X80.Log2, func(x *big.Float) *big.Float { return rf().Quo(refLn(x), refLn2) }, func(r *rand.Rand) X80 {
		return pick(r, func() X80 { return logUniform(r, -16440, 16383, true) },
			func() X80 { return X80One.Add(logUniform(r, -70, -1, false)) })
	}, 0.501},
	{"Log10", X80.Log10, func(x *big.Float) *big.Float { return rf().Quo(refLn(x), refLn10) }, func(r *rand.Rand) X80 {
		return pick(r, func() X80 { return logUniform(r, -16440, 16383, true) },
			func() X80 { return X80One.Add(logUniform(r, -70, -1, false)) })
	}, 0.501},
	{"Log1p", X80.Log1p, func(x *big.Float) *big.Float { return refLn(rf().Add(x, rInt(1))) }, func(r *rand.Rand) X80 {
		return pick(r, func() X80 { return logUniform(r, -70, 16383, true) },
			func() X80 { return logUniform(r, -70, -1, false) })
	}, 0.501},
	{"Sin", X80.Sin, func(x *big.Float) *big.Float { s, _ := refSinCos(x); return s }, func(r *rand.Rand) X80 {
		return pick(r, func() X80 { return below(r, 100, false) }, func() X80 { return logUniform(r, -30, 16383, false) })
	}, 0.501},
	{"Cos", X80.Cos, func(x *big.Float) *big.Float { _, c := refSinCos(x); return c }, func(r *rand.Rand) X80 {
		return pick(r, func() X80 { return below(r, 100, false) }, func() X80 { return logUniform(r, -30, 16383, false) })
	}, 0.501},
	{"Tan", X80.Tan, func(x *big.Float) *big.Float { s, c := refSinCos(x); return s.Quo(s, c) }, func(r *rand.Rand) X80 {
		return pick(r, func() X80 { return below(r, 100, false) }, func() X80 { return logUniform(r, -30, 16383, false) })
	}, 0.501},
	{"Atan", X80.Atan, refAtan, func(r *rand.Rand) X80 { return logUniform(r, -40, 200, false) }, 0.501},
	{"Asin", X80.Asin, func(x *big.Float) *big.Float {
		return refAtan(rf().Quo(x, rf().Sqrt(rf().Sub(rInt(1), rf().Mul(x, x)))))
	}, func(r *rand.Rand) X80 {
		return pick(r, func() X80 { return below(r, 1, false) }, func() X80 { return logUniform(r, -40, 0, false) })
	}, 0.501},
	{"Acos", X80.Acos, func(x *big.Float) *big.Float {
		asin := refAtan(rf().Quo(x, rf().Sqrt(rf().Sub(rInt(1), rf().Mul(x, x)))))
		return rf().Sub(rf().Quo(refPi(refPrec), rInt(2)), asin)
	}, func(r *rand.Rand) X80 {
		return pick(r, func() X80 { return below(r, 1, false) }, func() X80 { return logUniform(r, -40, 0, false) })
	}, 0.501},
	{"Sinh", X80.Sinh, func(x *big.Float) *big.Float {
		e := refExp(x)
		return e.Sub(e, rf().Quo(rInt(1), e)).Quo(e, rInt(2))
	}, func(r *rand.Rand) X80 {
		return pick(r, func() X80 { return below(r, 11355, false) }, func() X80 { return logUniform(r, -40, 6, false) })
	}, 0.501},
	{"Cosh", X80.Cosh, func(x *big.Float) *big.Float {
		e := refExp(x)
		return e.Add(e, rf().Quo(rInt(1), e)).Quo(e, rInt(2))
	}, func(r *rand.Rand) X80 {
		return pick(r, func() X80 { return below(r, 11355, false) }, func() X80 { return logUniform(r, -40, 6, false) })
	}, 0.501},
	{"Tanh", X80.Tanh, func(x *big.Float) *big.Float {
		e := refExp(rf().Mul(x, rInt(2)))
		return rf().Quo(rf().Sub(e, rInt(1)), rf().Add(e, rInt(1)))
	}, func(r *rand.Rand) X80 {
		return pick(r, func() X80 { return below(r, 40, false) }, func() X80 { return logUniform(r, -40, 6, false) })
	}, 0.501},
	{"Atanh", X80.Atanh, func(x *big.Float) *big.Float {
		l := refLn(rf().Quo(rf().Add(rInt(1), x), rf().Sub(rInt(1), x)))
		return l.Quo(l, rInt(2))
	}, func(r *rand.Rand) X80 {
		return pick(r, func() X80 { return below(r, 1, false) }, func() X80 { return logUniform(r, -40, 0, false) })
	}, 0.501},
}

func TestTranscendentalAccuracy(t *testing.T) {
	n := 3000
	if testing.Short() {
		n = 300
	}
	for _, tf := range transcendentals {
		t.Run(tf.name, func(t *testing.T) {
			r := rand.New(rand.NewPCG(uint64(len(tf.name)), 42))
			worst, worstX := 0.0, X80{}
			for range n {
				x := tf.gen(r)
				if x.IsZero() {
					continue
				}
				want := tf.ref(toBig(x))
				if !inNormalRange(want) {
					continue
				}
				if e := ulpErr(tf.f(x), want); e > worst {
					worst, worstX = e, x
				}
			}
			t.Logf("max error %.3f ulp at %s", worst, worstX.Internal())
			if worst > tf.tol {
				t.Errorf("max error %.3f ulp at %s exceeds %.1f ulp", worst, worstX.Internal(), tf.tol)
			}
		})
	}
}

// TestTwoOverPiTable checks the generated table against 2/pi computed from
// the reference pi.
func TestTwoOverPiTable(t *testing.T) {
	nbits := uint(64 * len(twoOverPi))
	twoOverPiRef := new(big.Float).SetPrec(nbits+64).Quo(big.NewFloat(2), refPiBig())
	want, _ := twoOverPiRef.SetMantExp(twoOverPiRef, int(nbits)).Int(nil)
	got := new(big.Int)
	for _, w := range twoOverPi {
		got.Lsh(got, 64).Or(got, new(big.Int).SetUint64(w))
	}
	if diff := new(big.Int).Sub(want, got); diff.Sign() != 0 && diff.CmpAbs(big.NewInt(1)) > 0 {
		t.Errorf("twoOverPi differs from 2/pi by %v units of the last word", diff)
	}
}
