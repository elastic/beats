package common

import "math"

// Entropy calculates the entropy of data
func Entropy(data []byte) float64 {
	cache := make(map[byte]int)
	for _, b := range data {
		if found, ok := cache[b]; ok {
			cache[b] = found + 1
		} else {
			cache[b] = 1
		}
	}

	result := 0.0
	length := len(data)
	for _, count := range cache {
		frequency := float64(count) / float64(length)
		result -= frequency * math.Log2(frequency)
	}
	return math.Round(result*100) / 100
}
