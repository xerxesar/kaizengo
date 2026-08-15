package inv

import "testing"

func TestConsumeLayersFIFO(t *testing.T) {
	layers := []CostLayer{
		{ID: "a", Remaining: 10, UnitCost: 2, ReceivedAt: "2026-01-01"},
		{ID: "b", Remaining: 10, UnitCost: 4, ReceivedAt: "2026-02-01"},
	}
	unit, left, err := ConsumeLayers(layers, 12, false)
	if err != nil {
		t.Fatal(err)
	}
	if unit != 2.3333 {
		t.Fatalf("fifo unit cost got %v want 2.3333", unit)
	}
	if left[0].Remaining != 0 || left[1].Remaining != 8 {
		t.Fatalf("remaining = %+v", left)
	}
}

func TestConsumeLayersLIFO(t *testing.T) {
	layers := []CostLayer{
		{ID: "a", Remaining: 10, UnitCost: 2, ReceivedAt: "2026-01-01"},
		{ID: "b", Remaining: 10, UnitCost: 4, ReceivedAt: "2026-02-01"},
	}
	unit, left, err := ConsumeLayers(layers, 6, true)
	if err != nil {
		t.Fatal(err)
	}
	if unit != 4 {
		t.Fatalf("lifo unit cost got %v", unit)
	}
	byID := map[string]float64{}
	for _, layer := range left {
		byID[layer.ID] = layer.Remaining
	}
	if byID["b"] != 4 || byID["a"] != 10 {
		t.Fatalf("remaining = %+v", left)
	}
}

func TestMovingAverage(t *testing.T) {
	got := movingAverage(10, 2, 10, 4)
	if got != 3 {
		t.Fatalf("got %v want 3", got)
	}
}
