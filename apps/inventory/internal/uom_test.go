package inv

import "testing"

func TestConvertQty(t *testing.T) {
	// 2 dozen → each (ratio 12 → 1)
	got, err := ConvertQty(2, 12, 1, 0.000001)
	if err != nil {
		t.Fatal(err)
	}
	if got != 24 {
		t.Fatalf("dozen to each: got %v want 24", got)
	}

	// 1500 g → kg (0.001 → 1)
	got, err = ConvertQty(1500, 0.001, 1, 0.000001)
	if err != nil {
		t.Fatal(err)
	}
	if got != 1.5 {
		t.Fatalf("g to kg: got %v want 1.5", got)
	}

	if _, err := ConvertQty(1, 0, 1, 0.000001); err == nil {
		t.Fatal("expected error for zero ratio")
	}
}

func TestRoundMoney(t *testing.T) {
	if got := RoundMoney(1.23456); got != 1.2346 {
		t.Fatalf("got %v", got)
	}
}
