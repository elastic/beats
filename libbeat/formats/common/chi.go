package common

import (
	"math"
)

// ChiSquare calculates the chi-squared distribution of data
func ChiSquare(data []byte) float64 {
	cache := make([]float64, 256)
	for _, b := range data {
		cache[b] = cache[b] + 1
	}

	result := 0.0
	length := len(data)
	perBin := float64(length) / float64(256) // expected count per bin
	if perBin == 0 {
		return 0.0
	}
	for _, count := range cache {
		a := count - perBin
		result += (a * a) / perBin
	}
	return math.Round(result*100) / 100
}
