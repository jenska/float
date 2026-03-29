package float

import "testing"

func BenchmarkX80_Add(b *testing.B) {
	a := X80One
	c := newFromHexString("40008000000000000000")
	for i := 0; i < b.N; i++ {
		a = a.Add(c)
	}
}

func BenchmarkX80_Mul(b *testing.B) {
	a := X80One
	c := newFromHexString("40008000000000000000")
	for i := 0; i < b.N; i++ {
		a = a.Mul(c)
	}
}

func BenchmarkX80_Div(b *testing.B) {
	a := X80Pi
	c := newFromHexString("40008000000000000000")
	for i := 0; i < b.N; i++ {
		a = a.Div(c)
	}
}

func BenchmarkX80_Sqrt(b *testing.B) {
	a := X80Pi
	for i := 0; i < b.N; i++ {
		a = a.Sqrt()
	}
}

func BenchmarkX80_Ln(b *testing.B) {
	a := X80E
	for i := 0; i < b.N; i++ {
		a = a.Ln()
	}
}

func BenchmarkX80_Atan(b *testing.B) {
	a := X80One
	for i := 0; i < b.N; i++ {
		a = a.Atan()
	}
}
