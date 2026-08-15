package inventory

import (
	"math"

	"kaizengo/packages/sdk-go/i18n"
)

const (
	defaultRounding = 0.000001
	moneyPlaces     = 10000.0
)

// ConvertQty converts qty from one UoM to another using category ratios to a shared base.
// ratio is "how many reference units in one of this UoM" (dozen ratio=12, gram ratio=0.001).
func ConvertQty(qty, fromRatio, toRatio, rounding float64) (float64, error) {
	if fromRatio <= 0 || toRatio <= 0 {
		return 0, i18n.Error("inventory.error.uom_ratio")
	}
	return RoundQty(qty*fromRatio/toRatio, rounding), nil
}

func RoundQty(v, rounding float64) float64 {
	if rounding <= 0 {
		rounding = defaultRounding
	}
	return math.Round(v/rounding) * rounding
}

func RoundMoney(v float64) float64 {
	return math.Round(v*moneyPlaces) / moneyPlaces
}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < defaultRounding
}
