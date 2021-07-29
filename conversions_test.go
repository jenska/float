package float

import (
	"math"
	"reflect"
	"testing"
)

func TestInt32ToFloatX80(t *testing.T) {
	tests := []struct {
		name string
		a    int32
		want X80
	}{
		{"zero", 0, X80Zero},
		{"1", 1, X80One},
		{"-1", -1, X80MinusOne},
		{"2", 2, newFromHexString("40008000000000000000")},
		{"-2", -2, newFromHexString("C0008000000000000000")},
		{"3", 3, newFromHexString("4000C000000000000000")},
		{"-3", -3, newFromHexString("C000C000000000000000")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Int32ToFloatX80(tt.a); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Int32ToFloatX80() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestX80_ToInt32(t *testing.T) {
	tests := []struct {
		name string
		a    X80
		want int32
	}{
		{"zero", X80Zero, 0},
		{"1", X80One, 1},
		{"-1", X80MinusOne, -1},
		{"2", newFromHexString("40008000000000000000"), 2},
		{"-2", newFromHexString("C0008000000000000000"), -2},
		{"3", newFromHexString("4000C000000000000000"), 3},
		{"-3", newFromHexString("C000C000000000000000"), -3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := tt.a
			if got := a.ToInt32(); got != tt.want {
				t.Errorf("X80.ToInt32() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestX80_ToInt64(t *testing.T) {
	tests := []struct {
		name string
		a    X80
		want int64
	}{
		{"zero", X80Zero, 0},
		{"1", X80One, 1},
		{"-1", X80MinusOne, -1},
		{"2", newFromHexString("40008000000000000000"), 2},
		{"-2", newFromHexString("C0008000000000000000"), -2},
		{"3", newFromHexString("4000C000000000000000"), 3},
		{"-3", newFromHexString("C000C000000000000000"), -3},
		{"0.5", newFromHexString("3FFE8000000000000000"), 0},
		{"-0.5", newFromHexString("BFFE8000000000000000"), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := tt.a
			if got := a.ToInt64(); got != tt.want {
				t.Errorf("X80.ToInt64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInt64ToFloatX80(t *testing.T) {
	tests := []struct {
		name string
		a    int64
		want X80
	}{
		{"zero", 0, X80Zero},
		{"1", 1, X80One},
		{"-1", -1, X80MinusOne},
		{"2", 2, newFromHexString("40008000000000000000")},
		{"-2", -2, newFromHexString("C0008000000000000000")},
		{"3", 3, newFromHexString("4000C000000000000000")},
		{"-3", -3, newFromHexString("C000C000000000000000")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Int64ToFloatX80(tt.a); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Int64ToFloatX80() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFloat64ToFloatX80(t *testing.T) {
	tests := []struct {
		name string
		a    float64
		want X80
	}{
		{"zero", 0.0, X80Zero},
		{"1", 1.0, X80One},
		{"-1", -1.0, X80MinusOne},
		{"2", 2.0, newFromHexString("40008000000000000000")},
		{"-2", -2.0, newFromHexString("C0008000000000000000")},
		{"3", 3.0, newFromHexString("4000C000000000000000")},
		{"-3", -3.0, newFromHexString("C000C000000000000000")},
		{"0.5", 0.5, newFromHexString("3FFE8000000000000000")},
		{"-0.5", -0.5, newFromHexString("BFFE8000000000000000")},
		{"0.25", 0.25, newFromHexString("3FFD8000000000000000")},
		{"-0.25", -0.25, newFromHexString("BFFD8000000000000000")},
		{"0.125", 0.125, newFromHexString("3FFC8000000000000000")},
		{"-0.125", -0.125, newFromHexString("BFFC8000000000000000")},
		{"0.123", 0.123, newFromHexString("3FFBFBE76C8B43958000")},
		{"-0.123", -0.123, newFromHexString("BFFBFBE76C8B43958000")},
		{"0.33333", 0.33333, newFromHexString("3FFDAAAA3AD18D25F000")},
		{"-0.33333", -0.33333, newFromHexString("BFFDAAAA3AD18D25F000")},
		{"inf+", math.Inf(1), X80InfPos},
		{"inf-", math.Inf(-1), X80InfNeg},
		{"pi", math.Pi, X80Pi},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Float64ToFloatX80(tt.a); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Float64ToFloatX80() = %v, want %v", got, tt.want)
			}
		})
	}
}
