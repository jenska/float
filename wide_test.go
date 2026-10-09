package float

import (
	"math"
	"math/big"
	"math/rand/v2"
	"testing"
)

func (a wide) big() *big.Float {
	z := new(big.Float).SetPrec(256)
	hi := new(big.Float).SetPrec(256).SetUint64(a.hi)
	z.SetUint64(a.lo).Add(z, hi.SetMantExp(hi, 64)).SetMantExp(z, a.exp-127)
	if a.neg {
		z.Neg(z)
	}
	return z
}

func randWide(r *rand.Rand, spread int) wide {
	return wide{r.IntN(2) == 0, r.IntN(2*spread+1) - spread, r.Uint64() | 1<<63, r.Uint64()}
}

// relErrLog2 returns log2 of the relative error of got against want.
func relErrLog2(got wide, want *big.Float) float64 {
	if want.Sign() == 0 {
		if got.isZero() {
			return math.Inf(-1)
		}
		return math.Inf(1)
	}
	d := new(big.Float).SetPrec(256).Sub(got.big(), want)
	f, _ := d.Quo(d, want).Float64()
	return math.Log2(math.Abs(f))
}

func TestWideOperations(t *testing.T) {
	r := rand.New(rand.NewPCG(61, 62))
	worst := map[string]float64{"add": math.Inf(-1), "mul": math.Inf(-1), "div": math.Inf(-1), "divInt": math.Inf(-1), "sqrt": math.Inf(-1)}
	for range 20000 {
		a, b := randWide(r, 40), randWide(r, 40)
		if r.IntN(4) == 0 { // close values: cancellation in sub
			b = a
			b.lo ^= r.Uint64() >> r.IntN(64)
			b.neg = !a.neg
		}
		ab, bb := a.big(), b.big()
		p := func() *big.Float { return new(big.Float).SetPrec(256) }
		for name, c := range map[string]struct {
			got  wide
			want *big.Float
		}{
			"add":    {a.add(b), p().Add(ab, bb)},
			"mul":    {a.mul(b), p().Mul(ab, bb)},
			"div":    {a.div(b), p().Quo(ab, bb)},
			"divInt": {a.divInt(7), p().Quo(ab, big.NewFloat(7))},
			"sqrt":   {a.abs().sqrt(), p().Sqrt(p().Abs(ab))},
		} {
			e := relErrLog2(c.got, c.want)
			if name == "add" && c.want.Sign() != 0 {
				// truncating the smaller operand costs at most one unit of the larger
				e -= float64(a.exp - max(c.want.MantExp(nil)-1, a.exp-200))
				e = min(e, relErrLog2(c.got, c.want))
			}
			worst[name] = max(worst[name], e)
		}
	}
	for name, limit := range map[string]float64{"add": -124, "mul": -124, "div": -124, "divInt": -124, "sqrt": -124} {
		t.Logf("%s: worst relative error 2^%.1f", name, worst[name])
		if worst[name] > limit {
			t.Errorf("%s: relative error 2^%.1f exceeds 2^%.0f", name, worst[name], limit)
		}
	}
}
