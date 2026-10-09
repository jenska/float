package float

import (
	"encoding/binary"
	"math/rand/v2"
	"testing"
)

// Benchmarks cycle through 1024 random operands of a given class, so neither
// a single input nor a chain of results that drifts into a special case
// (Inf, 0, NaN) dominates the measurement.

var (
	sinkX80  X80
	sinkBool bool
	sinkInt  int64
	sinkStr  string
)

const benchOperands = 1024

func operands(seed uint64, gen func(r *rand.Rand) X80) []X80 {
	r := rand.New(rand.NewPCG(seed, 99))
	xs := make([]X80, benchOperands)
	for i := range xs {
		xs[i] = gen(r)
	}
	return xs
}

// uniform returns operands in [lo, hi).
func uniform(seed uint64, lo, hi float64) []X80 {
	return operands(seed, func(r *rand.Rand) X80 {
		x := NewFromFloat64(lo + r.Float64()*(hi-lo))
		x.low |= r.Uint64() & 0x7FF // fill the bits below float64 precision
		return x
	})
}

// spread returns normal operands of both signs with exponents within ±spread.
func spread(seed uint64, spread int) []X80 {
	return operands(seed, func(r *rand.Rand) X80 { return randX80(r, spread) })
}

func benchUnary(b *testing.B, xs []X80, f func(X80) X80) {
	b.ReportAllocs()
	i := 0
	for b.Loop() {
		sinkX80 = f(xs[i&(benchOperands-1)])
		i++
	}
}

func benchBinary(b *testing.B, xs, ys []X80, f func(X80, X80) X80) {
	b.ReportAllocs()
	i := 0
	for b.Loop() {
		sinkX80 = f(xs[i&(benchOperands-1)], ys[i&(benchOperands-1)])
		i++
	}
}

func BenchmarkArithmetic(b *testing.B) {
	xs, ys := spread(1, 30), spread(2, 30)
	for _, c := range []struct {
		name string
		f    func(X80, X80) X80
	}{
		{"Add", X80.Add}, {"Sub", X80.Sub}, {"Mul", X80.Mul}, {"Div", X80.Div},
		{"SglMul", X80.SglMul}, {"SglDiv", X80.SglDiv}, {"Rem", X80.Rem}, {"Mod", X80.Mod},
	} {
		b.Run(c.name, func(b *testing.B) { benchBinary(b, xs, ys, c.f) })
	}
	b.Run("RemFar", func(b *testing.B) { benchBinary(b, spread(3, 2000), spread(4, 10), X80.Rem) })
	for _, c := range []struct {
		name string
		f    func(X80) X80
	}{
		{"Sqrt", func(x X80) X80 { return x.Abs().Sqrt() }}, {"RoundToInt", X80.RoundToInt},
		{"Trunc", X80.Trunc}, {"GetExp", X80.GetExp}, {"GetMan", X80.GetMan},
		{"Scale", func(x X80) X80 { return x.Scale(100) }},
		{"RoundToPrecision", func(x X80) X80 { return x.RoundToPrecision(64) }},
	} {
		b.Run(c.name, func(b *testing.B) { benchUnary(b, xs, c.f) })
	}
	b.Run("Unnormal", func(b *testing.B) {
		us := make([]X80, benchOperands)
		for i, x := range xs {
			us[i] = X80{high: x.high + 1, low: x.low >> 1}
		}
		benchBinary(b, us, ys, X80.Add)
	})
}

func BenchmarkCompare(b *testing.B) {
	xs, ys := spread(5, 3), spread(6, 3)
	for _, c := range []struct {
		name string
		f    func(X80, X80) bool
	}{{"Eq", X80.Eq}, {"Lt", X80.Lt}, {"Le", X80.Le}, {"Gt", X80.Gt}, {"LtQuiet", X80.LtQuiet}} {
		b.Run(c.name, func(b *testing.B) {
			i := 0
			for b.Loop() {
				sinkBool = c.f(xs[i&(benchOperands-1)], ys[i&(benchOperands-1)])
				i++
			}
		})
	}
}

func BenchmarkTranscendental(b *testing.B) {
	for _, c := range []struct {
		name string
		xs   []X80
		f    func(X80) X80
	}{
		{"Exp", uniform(10, -50, 50), X80.Exp},
		{"Exp2", uniform(11, -50, 50), X80.Exp2},
		{"Exp10", uniform(12, -20, 20), X80.Exp10},
		{"Expm1", uniform(13, -1, 1), X80.Expm1},
		{"Ln", uniform(14, 0, 1000), X80.Ln},
		{"Log2", uniform(15, 0, 1000), X80.Log2},
		{"Log10", uniform(16, 0, 1000), X80.Log10},
		{"Log1p", uniform(17, -0.5, 1), X80.Log1p},
		{"Sin/small", uniform(18, -0.78, 0.78), X80.Sin},
		{"Sin/medium", uniform(19, -1000, 1000), X80.Sin},
		{"Sin/huge", spread(20, 16000), X80.Sin},
		{"Cos/medium", uniform(21, -1000, 1000), X80.Cos},
		{"Tan/medium", uniform(22, -1000, 1000), X80.Tan},
		{"Atan", uniform(23, -10, 10), X80.Atan},
		{"Asin", uniform(24, -1, 1), X80.Asin},
		{"Acos", uniform(25, -1, 1), X80.Acos},
		{"Sinh", uniform(26, -20, 20), X80.Sinh},
		{"Cosh", uniform(27, -20, 20), X80.Cosh},
		{"Tanh", uniform(28, -5, 5), X80.Tanh},
		{"Atanh", uniform(29, -1, 1), X80.Atanh},
	} {
		b.Run(c.name, func(b *testing.B) { benchUnary(b, c.xs, c.f) })
	}
	b.Run("Sincos/medium", func(b *testing.B) {
		xs := uniform(30, -1000, 1000)
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			sinkX80, _ = xs[i&(benchOperands-1)].Sincos()
			i++
		}
	})
}

func BenchmarkConversion(b *testing.B) {
	xs := spread(40, 60)
	small := uniform(41, -30000, 30000)
	f64 := make([]uint64, benchOperands)
	for i, x := range xs {
		f64[i] = x.ToFloat64Bits()
	}
	run := func(name string, f func(i int)) {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			i := 0
			for b.Loop() {
				f(i & (benchOperands - 1))
				i++
			}
		})
	}
	run("ToInt64", func(i int) { sinkInt = xs[i].ToInt64() })
	run("ToInt32", func(i int) { sinkInt = int64(small[i].ToInt32()) })
	run("ToInt16", func(i int) { sinkInt = int64(small[i].ToInt16()) })
	run("Int64ToFloatX80", func(i int) { sinkX80 = Int64ToFloatX80(int64(f64[i])) })
	run("ToFloat64", func(i int) { sinkInt = int64(xs[i].ToFloat64Bits()) })
	run("ToFloat32", func(i int) { sinkInt = int64(xs[i].ToFloat32Bits()) })
	run("FromFloat64", func(i int) { sinkX80 = NewFromFloat64Bits(f64[i]) })
	run("Bytes96", func(i int) { sinkX80 = NewFromBytes96(xs[i].Bytes96(binary.BigEndian), binary.BigEndian) })
}

func BenchmarkDecimal(b *testing.B) {
	xs := spread(50, 60)
	strs := make([]string, benchOperands)
	for i, x := range xs {
		strs[i] = x.String()
	}
	run := func(name string, f func(i int)) {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			i := 0
			for b.Loop() {
				f(i & (benchOperands - 1))
				i++
			}
		})
	}
	run("String", func(i int) { sinkStr = xs[i].String() })
	run("Format_e", func(i int) { sinkStr = xs[i].Format('e', 10) })
	huge := spread(51, 16000)
	run("String/huge", func(i int) { sinkStr = huge[i].String() })
	run("Parse", func(i int) { sinkX80, _ = Parse(strs[i]) })
	run("Pow10", func(i int) { sinkX80 = Pow10(i%600 - 300) })
	run("ConstValue", func(i int) { sinkX80 = Constant(i % 7).Value() })
}
