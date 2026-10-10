package float

// Env is a floating-point environment: the rounding control that operations
// use and the exception flags they raise, like the x87 control and status
// words or the 68881/68882 FPCR and FPSR.  Every operation that rounds or
// raises exceptions is a method of Env; "the current rounding mode" and
// similar phrases in their descriptions refer to the fields of the Env.
//
// An Env must not be used by several goroutines at once.  Concurrent code
// gives each goroutine its own Env; separate Envs share no state.
//
// The zero Env rounds to nearest even at full 80-bit precision, detects
// tininess after rounding and returns X80NaN for invalid operations.
//
// The X80 methods and package-level functions such as X80.Add and Pow10 use
// an Env with the settings of the package-level variables RoundingMode,
// RoundingPrecision, DetectTininess and DefaultNaN, which reports exceptions
// through Raise.  They are not safe for concurrent use.
type Env struct {
	RoundingMode      int // RoundNearestEven, RoundToZero, RoundDown or RoundUp
	RoundingPrecision int // 32, 64 or 80; other values select 80
	DetectTininess    int // TininessAfterRounding or TininessBeforeRounding

	// DefaultNaN is the quiet NaN returned by invalid operations when no
	// operand is a NaN.  The zero value selects X80NaN.
	DefaultNaN X80

	// Exception accumulates the exception flags raised in this Env until
	// they are cleared.
	Exception int

	// Handler, if not nil, is called with the flags of every raised
	// exception.
	Handler ExceptionHandler
}

// pkgEnv returns an Env with the package-level settings that reports
// exceptions through the package-level Raise.
func pkgEnv() *Env {
	return &Env{
		RoundingMode:      RoundingMode,
		RoundingPrecision: RoundingPrecision,
		DetectTininess:    DetectTininess,
		DefaultNaN:        DefaultNaN,
		Handler:           Raise,
	}
}

// Raise sets the exception flags x in e.Exception and calls e.Handler.
func (e *Env) Raise(x int) {
	e.Exception |= x
	if e.Handler != nil {
		e.Handler(x)
	}
}

// quiet returns an Env with the settings of e that records exceptions only
// in its own Exception field, for operations that combine the exceptions of
// intermediate steps before raising them in e.
func (e *Env) quiet() Env {
	return Env{
		RoundingMode:      e.RoundingMode,
		RoundingPrecision: e.RoundingPrecision,
		DetectTininess:    e.DetectTininess,
		DefaultNaN:        e.DefaultNaN,
	}
}

func (e *Env) defaultNaN() X80 {
	if e.DefaultNaN == (X80{}) {
		return X80NaN
	}
	return e.DefaultNaN
}
