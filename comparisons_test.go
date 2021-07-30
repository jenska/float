package float

import (
	"testing"
)

func TestX80_Eq(t *testing.T) {
	one := X80One
	two := newFromHexString("40008000000000000000")

	a := one.Add(one)
	if !a.Eq(two) {
		t.Errorf("Eq failed: compared %v and %v, expected equal", a, two)
	}

	if a.Eq(one) {
		t.Error("Eq failed")
	}

	if a.Eq(X80NaN) {
		t.Error("Eq failed")
	}

	if X80InfNeg.Eq(X80InfPos) {
		t.Error("Eq failed")
	}
}

func TestX80_Le(t *testing.T) {
	one := X80One
	two := newFromHexString("40008000000000000000")

	a := one.Add(one)
	if !a.Le(two) {
		t.Errorf("Le failed: compared %v and %v, expected equal", a, two)
	}

	if a.Le(one) {
		t.Error("LE failed")
	}

	if a.Le(X80NaN) {
		t.Error("LE failed")
	}

	if X80InfNeg.Gt(X80InfPos) {
		t.Error("LE failed")
	}
}

func TestX80_Lt(t *testing.T) {
	one := X80One
	two := newFromHexString("40008000000000000000")

	a := one.Add(one)
	if !a.Ge(two) {
		t.Errorf("Ge failed: compared %v and %v, expected equal", a, two)
	}

	if a.Lt(one) {
		t.Errorf("Eq failed")
	}

	if a.Lt(X80NaN) {
		t.Errorf("Eq failed")
	}

	if X80InfNeg.Ge(X80InfPos) {
		t.Errorf("Eq failed")
	}
}

func TestX80_EqSignaling(t *testing.T) {
	one := X80One
	two := newFromHexString("40008000000000000000")

	a := one.Add(one)
	if !a.EqSignaling(two) {
		t.Errorf("Eq failed: compared %v and %v, expected equal", a, two)
	}

	if a.EqSignaling(one) {
		t.Error("Eq failed")
	}

	if a.EqSignaling(X80NaN) {
		t.Error("Eq failed")
	}

	if X80InfNeg.EqSignaling(X80InfPos) {
		t.Error("Eq failed")
	}
}

func TestX80_LtQuiet(t *testing.T) {
	one := X80One
	two := newFromHexString("40008000000000000000")

	a := one.Add(one)
	if !a.GeQuiet(two) {
		t.Errorf("Ge failed: compared %v and %v, expected equal", a, two)
	}

	if a.LtQuiet(one) {
		t.Errorf("LtQuiet failed")
	}

	if a.LtQuiet(X80NaN) {
		t.Errorf("LtQuiet failed")
	}

	if X80InfNeg.GeQuiet(X80InfPos) {
		t.Errorf("LtQuiet failed")
	}
}

func TestX80_LeQuiet(t *testing.T) {
	one := X80One
	two := newFromHexString("40008000000000000000")

	a := one.Add(one)
	if !a.LeQuiet(two) {
		t.Errorf("Le failed: compared %v and %v, expected equal", a, two)
	}

	if a.LeQuiet(one) {
		t.Error("LE failed")
	}

	if a.LeQuiet(X80NaN) {
		t.Error("LE failed")
	}

	if X80InfNeg.GtQuiet(X80InfPos) {
		t.Error("LE failed")
	}
}
