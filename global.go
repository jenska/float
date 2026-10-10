package float

// The X80 methods and package-level functions below use the package-level
// environment; see [Env].

// NewFromFloat64 is [Env.NewFromFloat64] with the package-level environment.
func NewFromFloat64(a float64) X80 { return pkgEnv().NewFromFloat64(a) }

// Float32ToFloatX80 is [Env.Float32ToFloatX80] with the package-level environment.
func Float32ToFloatX80(a float32) X80 { return pkgEnv().Float32ToFloatX80(a) }

// Float64ToFloatX80 is [Env.Float64ToFloatX80] with the package-level environment.
func Float64ToFloatX80(a float64) X80 { return pkgEnv().Float64ToFloatX80(a) }

// NewFromFloat32Bits is [Env.NewFromFloat32Bits] with the package-level environment.
func NewFromFloat32Bits(b uint32) X80 { return pkgEnv().NewFromFloat32Bits(b) }

// NewFromFloat64Bits is [Env.NewFromFloat64Bits] with the package-level environment.
func NewFromFloat64Bits(b uint64) X80 { return pkgEnv().NewFromFloat64Bits(b) }

// ToInt32 is [Env.ToInt32] with the package-level environment.
func (a X80) ToInt32() int32 { return pkgEnv().ToInt32(a) }

// ToInt32RoundZero is [Env.ToInt32RoundZero] with the package-level environment.
func (a X80) ToInt32RoundZero() int32 { return pkgEnv().ToInt32RoundZero(a) }

// ToInt16 is [Env.ToInt16] with the package-level environment.
func (a X80) ToInt16() int16 { return pkgEnv().ToInt16(a) }

// ToInt8 is [Env.ToInt8] with the package-level environment.
func (a X80) ToInt8() int8 { return pkgEnv().ToInt8(a) }

// ToInt64 is [Env.ToInt64] with the package-level environment.
func (a X80) ToInt64() int64 { return pkgEnv().ToInt64(a) }

// ToInt64RoundZero is [Env.ToInt64RoundZero] with the package-level environment.
func (a X80) ToInt64RoundZero() int64 { return pkgEnv().ToInt64RoundZero(a) }

// ToFloat32 is [Env.ToFloat32] with the package-level environment.
func (a X80) ToFloat32() float32 { return pkgEnv().ToFloat32(a) }

// ToFloat32Bits is [Env.ToFloat32Bits] with the package-level environment.
func (a X80) ToFloat32Bits() uint32 { return pkgEnv().ToFloat32Bits(a) }

// ToFloat64 is [Env.ToFloat64] with the package-level environment.
func (a X80) ToFloat64() float64 { return pkgEnv().ToFloat64(a) }

// ToFloat64Bits is [Env.ToFloat64Bits] with the package-level environment.
func (a X80) ToFloat64Bits() uint64 { return pkgEnv().ToFloat64Bits(a) }

// RoundToInt is [Env.RoundToInt] with the package-level environment.
func (a X80) RoundToInt() X80 { return pkgEnv().RoundToInt(a) }

// Add is [Env.Add] with the package-level environment.
func (a X80) Add(b X80) X80 { return pkgEnv().Add(a, b) }

// Sub is [Env.Sub] with the package-level environment.
func (a X80) Sub(b X80) X80 { return pkgEnv().Sub(a, b) }

// Mul is [Env.Mul] with the package-level environment.
func (a X80) Mul(b X80) X80 { return pkgEnv().Mul(a, b) }

// SglMul is [Env.SglMul] with the package-level environment.
func (a X80) SglMul(b X80) X80 { return pkgEnv().SglMul(a, b) }

// Div is [Env.Div] with the package-level environment.
func (a X80) Div(b X80) X80 { return pkgEnv().Div(a, b) }

// SglDiv is [Env.SglDiv] with the package-level environment.
func (a X80) SglDiv(b X80) X80 { return pkgEnv().SglDiv(a, b) }

// Rem is [Env.Rem] with the package-level environment.
func (a X80) Rem(b X80) X80 { return pkgEnv().Rem(a, b) }

// RemQuo is [Env.RemQuo] with the package-level environment.
func (a X80) RemQuo(b X80) (z X80, quo int) { return pkgEnv().RemQuo(a, b) }

// Mod is [Env.Mod] with the package-level environment.
func (a X80) Mod(b X80) X80 { return pkgEnv().Mod(a, b) }

// ModQuo is [Env.ModQuo] with the package-level environment.
func (a X80) ModQuo(b X80) (z X80, quo int) { return pkgEnv().ModQuo(a, b) }

// Scale is [Env.Scale] with the package-level environment.
func (a X80) Scale(n int) X80 { return pkgEnv().Scale(a, n) }

// GetExp is [Env.GetExp] with the package-level environment.
func (a X80) GetExp() X80 { return pkgEnv().GetExp(a) }

// GetMan is [Env.GetMan] with the package-level environment.
func (a X80) GetMan() X80 { return pkgEnv().GetMan(a) }

// RoundToPrecision is [Env.RoundToPrecision] with the package-level environment.
func (a X80) RoundToPrecision(prec int) X80 { return pkgEnv().RoundToPrecision(a, prec) }

// Trunc is [Env.Trunc] with the package-level environment.
func (a X80) Trunc() X80 { return pkgEnv().Trunc(a) }

// Sqrt is [Env.Sqrt] with the package-level environment.
func (a X80) Sqrt() X80 { return pkgEnv().Sqrt(a) }

// Eq is [Env.Eq] with the package-level environment.
func (a X80) Eq(b X80) bool { return pkgEnv().Eq(a, b) }

// Gt is [Env.Gt] with the package-level environment.
func (a X80) Gt(b X80) bool { return pkgEnv().Gt(a, b) }

// Le is [Env.Le] with the package-level environment.
func (a X80) Le(b X80) bool { return pkgEnv().Le(a, b) }

// Ge is [Env.Ge] with the package-level environment.
func (a X80) Ge(b X80) bool { return pkgEnv().Ge(a, b) }

// Lt is [Env.Lt] with the package-level environment.
func (a X80) Lt(b X80) bool { return pkgEnv().Lt(a, b) }

// EqSignaling is [Env.EqSignaling] with the package-level environment.
func (a X80) EqSignaling(b X80) bool { return pkgEnv().EqSignaling(a, b) }

// GtQuiet is [Env.GtQuiet] with the package-level environment.
func (a X80) GtQuiet(b X80) bool { return pkgEnv().GtQuiet(a, b) }

// LeQuiet is [Env.LeQuiet] with the package-level environment.
func (a X80) LeQuiet(b X80) bool { return pkgEnv().LeQuiet(a, b) }

// GeQuiet is [Env.GeQuiet] with the package-level environment.
func (a X80) GeQuiet(b X80) bool { return pkgEnv().GeQuiet(a, b) }

// LtQuiet is [Env.LtQuiet] with the package-level environment.
func (a X80) LtQuiet(b X80) bool { return pkgEnv().LtQuiet(a, b) }

// Value is [Env.Constant] with the package-level environment.
func (c Constant) Value() X80 { return pkgEnv().Constant(c) }

// Pow10 is [Env.Pow10] with the package-level environment.
func Pow10(n int) X80 { return pkgEnv().Pow10(n) }

// Parse is [Env.Parse] with the package-level environment.
func Parse(s string) (X80, error) { return pkgEnv().Parse(s) }

// Ln is [Env.Ln] with the package-level environment.
func (a X80) Ln() X80 { return pkgEnv().Ln(a) }

// Log2 is [Env.Log2] with the package-level environment.
func (a X80) Log2() X80 { return pkgEnv().Log2(a) }

// Log10 is [Env.Log10] with the package-level environment.
func (a X80) Log10() X80 { return pkgEnv().Log10(a) }

// Log1p is [Env.Log1p] with the package-level environment.
func (a X80) Log1p() X80 { return pkgEnv().Log1p(a) }

// Exp is [Env.Exp] with the package-level environment.
func (a X80) Exp() X80 { return pkgEnv().Exp(a) }

// Exp2 is [Env.Exp2] with the package-level environment.
func (a X80) Exp2() X80 { return pkgEnv().Exp2(a) }

// Exp10 is [Env.Exp10] with the package-level environment.
func (a X80) Exp10() X80 { return pkgEnv().Exp10(a) }

// Expm1 is [Env.Expm1] with the package-level environment.
func (a X80) Expm1() X80 { return pkgEnv().Expm1(a) }

// Sinh is [Env.Sinh] with the package-level environment.
func (a X80) Sinh() X80 { return pkgEnv().Sinh(a) }

// Cosh is [Env.Cosh] with the package-level environment.
func (a X80) Cosh() X80 { return pkgEnv().Cosh(a) }

// Tanh is [Env.Tanh] with the package-level environment.
func (a X80) Tanh() X80 { return pkgEnv().Tanh(a) }

// Atanh is [Env.Atanh] with the package-level environment.
func (a X80) Atanh() X80 { return pkgEnv().Atanh(a) }

// Atan is [Env.Atan] with the package-level environment.
func (a X80) Atan() X80 { return pkgEnv().Atan(a) }

// Asin is [Env.Asin] with the package-level environment.
func (a X80) Asin() X80 { return pkgEnv().Asin(a) }

// Acos is [Env.Acos] with the package-level environment.
func (a X80) Acos() X80 { return pkgEnv().Acos(a) }

// Sin is [Env.Sin] with the package-level environment.
func (a X80) Sin() X80 { return pkgEnv().Sin(a) }

// Cos is [Env.Cos] with the package-level environment.
func (a X80) Cos() X80 { return pkgEnv().Cos(a) }

// Sincos is [Env.Sincos] with the package-level environment.
func (a X80) Sincos() (sin, cos X80) { return pkgEnv().Sincos(a) }

// Tan is [Env.Tan] with the package-level environment.
func (a X80) Tan() X80 { return pkgEnv().Tan(a) }
