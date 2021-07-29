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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Add(tt.b); !got.Eq(tt.want) {
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Sub(tt.b); !got.Eq(tt.want) {
				t.Errorf("X80.Sub() = %v, want %v", got, tt.want)
			}
		})
	}
}
