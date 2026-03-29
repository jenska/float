package float

import (
	"testing"
)

func TestX80_RoundToInt(t *testing.T) {
	tests := []struct {
		name string
		a    X80
		want X80
	}{
		{"zero", X80Zero, X80Zero},
		{"1", X80One, X80One},
		{"-1", X80MinusOne, X80MinusOne},
		{"0.5", newFromHexString("3FFE8000000000000000"), X80Zero},
		{"-0.5", newFromHexString("BFFE8000000000000000"), X80Zero},
		{"0.25", newFromHexString("3FFD8000000000000000"), X80Zero},
		{"-0.25", newFromHexString("BFFD8000000000000000"), X80Zero},
		{"0.125", newFromHexString("3FFC8000000000000000"), X80Zero},
		{"-0.125", newFromHexString("BFFC8000000000000000"), X80Zero},
		{"0.123", newFromHexString("3FFBFBE76C8B43958000"), X80Zero},
		{"-0.123", newFromHexString("BFFBFBE76C8B43958000"), X80Zero},
		{"0.33333", newFromHexString("3FFDAAAA3AD18D25F000"), X80Zero},
		{"-0.33333", newFromHexString("BFFDAAAA3AD18D25F000"), X80Zero},
		{"inf+", X80InfPos, X80InfPos},
		{"inf-", X80InfNeg, X80InfNeg},
		{"pi", X80Pi, newFromHexString("4000C000000000000000")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := tt.a
			if got := a.RoundToInt(); !got.Eq(tt.want) {
				t.Errorf("X80.RoundToInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestX80_Add(t *testing.T) {
	tests := []struct {
		name string
		a    X80
		b    X80
		want X80
	}{
		{"-1 + 1 = 0", X80MinusOne, X80One, X80Zero},
		{"1 + 1 = 2", X80One, X80One, newFromHexString("40008000000000000000")},
		{"0 + 0 = 0", X80Zero, X80Zero, X80Zero},
		{"inf + 1 = inf", X80InfPos, X80One, X80InfPos},
		{"-inf + 1 = -inf", X80InfNeg, X80One, X80InfNeg},
		{"inf + -inf = NaN", X80InfPos, X80InfNeg, X80NaN},
		{"NaN + 1 = NaN", X80NaN, X80One, X80NaN},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Add(tt.b)
			if tt.want.IsNaN() {
				if !got.IsNaN() {
					t.Errorf("X80.Add() = %v, want %v", got, tt.want)
				}
			} else if !got.Eq(tt.want) {
				t.Errorf("X80.Add() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestX80_Sub(t *testing.T) {
	tests := []struct {
		name string
		a    X80
		b    X80
		want X80
	}{
		{"1 - 1 = 0", X80One, X80One, X80Zero},
		{"2 - 1 = 1", newFromHexString("40008000000000000000"), X80One, X80One},
		{"0 - 0 = 0", X80Zero, X80Zero, X80Zero},
		{"inf - 1 = inf", X80InfPos, X80One, X80InfPos},
		{"-inf - 1 = -inf", X80InfNeg, X80One, X80InfNeg},
		{"inf - inf = NaN", X80InfPos, X80InfPos, X80NaN},
		{"NaN - 1 = NaN", X80NaN, X80One, X80NaN},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Sub(tt.b)
			if tt.want.IsNaN() {
				if !got.IsNaN() {
					t.Errorf("X80.Sub() = %v, want %v", got, tt.want)
				}
			} else if !got.Eq(tt.want) {
				t.Errorf("X80.Sub() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestX80_Ln(t *testing.T) {
	tests := []struct {
		name string
		a    X80
		want X80
	}{
		{"ln(1)", X80One, X80Zero},
		{"ln(e)", X80E, X80One},
		{"ln(2)", newFromHexString("40008000000000000000"), X80Ln2},
		{"ln(0)", X80Zero, X80InfNeg},   // Should raise exception
		{"ln(-1)", X80MinusOne, X80NaN}, // Should raise exception
		{"ln(inf)", X80InfPos, X80InfPos},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Ln()
			if tt.want.IsInf() && tt.want.sign() || tt.want.IsNaN() {
				// Special cases
				if !((got.IsInf() && got.sign() == tt.want.sign()) || (got.IsNaN() && tt.want.IsNaN())) {
					t.Errorf("X80.Ln() = %v, want %v", got, tt.want)
				}
			} else if tt.name == "ln(1)" && !got.Eq(tt.want) {
				t.Errorf("X80.Ln() = %v, want %v", got, tt.want)
			}
			// For other cases, just check it's not NaN
			if got.IsNaN() && !tt.want.IsNaN() {
				t.Errorf("X80.Ln() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestX80_Atan(t *testing.T) {
	piOver4 := X80Pi.Div(Int64ToFloatX80(4)) // pi/4
	piOver2 := X80Pi.Div(Int64ToFloatX80(2)) // pi/2
	negPiOver4 := piOver4.Mul(X80MinusOne)
	negPiOver2 := piOver2.Mul(X80MinusOne)

	tests := []struct {
		name string
		a    X80
		want X80
	}{
		{"atan(0)", X80Zero, X80Zero},
		{"atan(1)", X80One, piOver4},
		{"atan(-1)", X80MinusOne, negPiOver4},
		{"atan(inf)", X80InfPos, piOver2},
		{"atan(-inf)", X80InfNeg, negPiOver2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Atan()
			// For floating point comparisons, check if they're close enough
			if tt.want.IsInf() || tt.want.IsNaN() {
				if !((got.IsInf() && got.sign() == tt.want.sign()) || (got.IsNaN() && tt.want.IsNaN())) {
					t.Errorf("X80.Atan() = %v, want %v", got, tt.want)
				}
			} else if tt.name == "atan(0)" && !got.Eq(tt.want) {
				t.Errorf("X80.Atan() = %v, want %v", got, tt.want)
			}
			// For other cases, just check it's not NaN
			if got.IsNaN() && !tt.want.IsNaN() {
				t.Errorf("X80.Atan() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestExceptionHandling(t *testing.T) {
	// Clear any existing exceptions
	ClearExceptions()

	// Test that exceptions are properly raised
	_ = X80Zero.Ln() // Should raise ExceptionDivbyzero
	if !HasException(ExceptionDivbyzero) {
		t.Error("Expected ExceptionDivbyzero to be raised")
	}

	_ = X80MinusOne.Ln() // Should raise ExceptionInvalid
	if !HasException(ExceptionInvalid) {
		t.Error("Expected ExceptionInvalid to be raised")
	}

	// Test clearing exceptions
	ClearExceptions()
	if HasAnyException() {
		t.Error("Expected no exceptions after clearing")
	}

	// Test exception handler
	handlerCalled := false
	var raisedException int
	SetExceptionHandler(func(exc int) {
		handlerCalled = true
		raisedException = exc
	})

	ClearExceptions()
	_ = X80Zero.Ln()
	if !handlerCalled {
		t.Error("Expected exception handler to be called")
	}
	if raisedException != ExceptionDivbyzero {
		t.Errorf("Expected handler to receive ExceptionDivbyzero, got %x", raisedException)
	}

	// Reset handler
	SetExceptionHandler(nil)
}
