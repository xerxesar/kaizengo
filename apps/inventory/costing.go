package inventory

import (
	"fmt"
	"sort"
)

// CostLayer is a remaining valuation slice for FIFO/LIFO consumption.
type CostLayer struct {
	ID         string
	Remaining  float64
	UnitCost   float64
	ReceivedAt string
}

// ConsumeLayers pulls qty from layers in FIFO (oldest first) or LIFO (newest first).
// Returns the average unit cost of consumed quantity and the mutated remaining slices.
func ConsumeLayers(layers []CostLayer, qty float64, lifo bool) (unitCost float64, remaining []CostLayer, err error) {
	if qty < 0 {
		return 0, layers, fmt.Errorf("quantity to consume must be >= 0")
	}
	if almostEqual(qty, 0) {
		return 0, layers, nil
	}
	ordered := append([]CostLayer(nil), layers...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].ReceivedAt == ordered[j].ReceivedAt {
			return ordered[i].ID < ordered[j].ID
		}
		if lifo {
			return ordered[i].ReceivedAt > ordered[j].ReceivedAt
		}
		return ordered[i].ReceivedAt < ordered[j].ReceivedAt
	})

	need := qty
	var costSum float64
	var taken float64
	for i := range ordered {
		if need <= 0 || ordered[i].Remaining <= 0 {
			continue
		}
		take := ordered[i].Remaining
		if take > need {
			take = need
		}
		costSum += take * ordered[i].UnitCost
		ordered[i].Remaining = RoundQty(ordered[i].Remaining-take, defaultRounding)
		need = RoundQty(need-take, defaultRounding)
		taken += take
	}
	if taken <= 0 {
		return 0, ordered, nil
	}
	return RoundMoney(costSum / taken), ordered, nil
}

func movingAverage(onHand, onHandCost, inQty, inCost float64) float64 {
	total := onHand + inQty
	if total <= 0 {
		return RoundMoney(inCost)
	}
	return RoundMoney((onHand*onHandCost + inQty*inCost) / total)
}
