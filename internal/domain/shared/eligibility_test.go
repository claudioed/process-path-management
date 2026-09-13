package shared

import "testing"

func TestEligibility_ZeroValue_IsPermissive(t *testing.T) {
	var e Eligibility
	if e.MaxUnitsPerLine() != nil {
		t.Fatalf("want nil (unbounded) MaxUnitsPerLine, got %v", *e.MaxUnitsPerLine())
	}
	if len(e.RequiredProductAttributes()) != 0 {
		t.Fatalf("want empty RequiredProductAttributes, got %v", e.RequiredProductAttributes())
	}
	if len(e.ExcludedProductAttributes()) != 0 {
		t.Fatalf("want empty ExcludedProductAttributes, got %v", e.ExcludedProductAttributes())
	}
	if e.NonSortable() {
		t.Fatal("want NonSortable=false by default")
	}
}

func TestNewEligibility_RoundTripsAllFields(t *testing.T) {
	maxUnits := 1
	e := NewEligibility(&maxUnits, []string{"giftWrap"}, []string{"hazmat", "fragile"}, true)
	if e.MaxUnitsPerLine() == nil || *e.MaxUnitsPerLine() != 1 {
		t.Fatalf("want MaxUnitsPerLine=1, got %v", e.MaxUnitsPerLine())
	}
	if len(e.RequiredProductAttributes()) != 1 || e.RequiredProductAttributes()[0] != "giftWrap" {
		t.Fatalf("unexpected RequiredProductAttributes: %v", e.RequiredProductAttributes())
	}
	if len(e.ExcludedProductAttributes()) != 2 {
		t.Fatalf("unexpected ExcludedProductAttributes: %v", e.ExcludedProductAttributes())
	}
	if !e.NonSortable() {
		t.Fatal("want NonSortable=true")
	}
}

func TestNewEligibility_MaxUnitsPerLine_IsDefensivelyCopied(t *testing.T) {
	maxUnits := 1
	e := NewEligibility(&maxUnits, nil, nil, false)
	maxUnits = 99
	if *e.MaxUnitsPerLine() != 1 {
		t.Fatal("expected the value object's internal state to be unaffected by mutating the caller's pointer target")
	}

	got := e.MaxUnitsPerLine()
	*got = 42
	if *e.MaxUnitsPerLine() != 1 {
		t.Fatal("expected MaxUnitsPerLine() to return a defensive copy, not a live pointer into internal state")
	}
}

func TestEligibility_RequiredProductAttributes_ReturnsDefensiveCopy(t *testing.T) {
	e := NewEligibility(nil, []string{"giftWrap"}, nil, false)
	got := e.RequiredProductAttributes()
	got[0] = "mutated"
	if e.RequiredProductAttributes()[0] != "giftWrap" {
		t.Fatal("expected internal state to be unaffected by mutating the returned slice")
	}
}

func TestEligibility_ExcludedProductAttributes_ReturnsDefensiveCopy(t *testing.T) {
	e := NewEligibility(nil, nil, []string{"hazmat"}, false)
	got := e.ExcludedProductAttributes()
	got[0] = "mutated"
	if e.ExcludedProductAttributes()[0] != "hazmat" {
		t.Fatal("expected internal state to be unaffected by mutating the returned slice")
	}
}

func TestEligibility_Equal_ZeroValues(t *testing.T) {
	if !(Eligibility{}).Equal(Eligibility{}) {
		t.Fatal("want two zero-value Eligibility to be equal")
	}
}

func TestEligibility_Equal_SameContent(t *testing.T) {
	maxUnits := 1
	a := NewEligibility(&maxUnits, []string{"giftWrap"}, []string{"hazmat"}, true)
	b := NewEligibility(&maxUnits, []string{"giftWrap"}, []string{"hazmat"}, true)
	if !a.Equal(b) {
		t.Fatalf("want equal, got a=%+v b=%+v", a, b)
	}
}

func TestEligibility_Equal_DifferentMaxUnitsPerLine(t *testing.T) {
	one, two := 1, 2
	a := NewEligibility(&one, nil, nil, false)
	b := NewEligibility(&two, nil, nil, false)
	if a.Equal(b) {
		t.Fatal("want not equal when MaxUnitsPerLine differs")
	}
}

func TestEligibility_Equal_NilVsSetMaxUnitsPerLine(t *testing.T) {
	one := 1
	a := NewEligibility(nil, nil, nil, false)
	b := NewEligibility(&one, nil, nil, false)
	if a.Equal(b) || b.Equal(a) {
		t.Fatal("want not equal when one side is unbounded and the other is not")
	}
}

func TestEligibility_Equal_DifferentNonSortable(t *testing.T) {
	a := NewEligibility(nil, nil, nil, false)
	b := NewEligibility(nil, nil, nil, true)
	if a.Equal(b) {
		t.Fatal("want not equal when NonSortable differs")
	}
}

func TestEligibility_Equal_DifferentRequiredProductAttributes(t *testing.T) {
	a := NewEligibility(nil, []string{"giftWrap"}, nil, false)
	b := NewEligibility(nil, []string{"fragile"}, nil, false)
	if a.Equal(b) {
		t.Fatal("want not equal when RequiredProductAttributes differ")
	}
}

func TestEligibility_Equal_DifferentExcludedProductAttributes(t *testing.T) {
	a := NewEligibility(nil, nil, []string{"hazmat"}, false)
	b := NewEligibility(nil, nil, []string{"fragile"}, false)
	if a.Equal(b) {
		t.Fatal("want not equal when ExcludedProductAttributes differ")
	}
}

func TestEligibility_Equal_DifferentAttributeCount(t *testing.T) {
	a := NewEligibility(nil, []string{"giftWrap"}, nil, false)
	b := NewEligibility(nil, []string{"giftWrap", "fragile"}, nil, false)
	if a.Equal(b) {
		t.Fatal("want not equal when RequiredProductAttributes length differs")
	}
}
