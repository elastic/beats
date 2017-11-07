package common

import "math"

// DefaultDecimalPlacesCount is the default number of decimal places
const DefaultDecimalPlacesCount = 4

// Round rounds the given float64 value and ensures that it has a maximum of
// "precision" decimal places.
func Round(val float64, precision int64) (newVal float64) {
	var round float64
	pow := math.Pow(10, float64(precision))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= 0.5 {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	newVal = round / pow
	return
}
