package qcgen

import "math/rand"

func GenBool(rng *rand.Rand) bool {
	return (rng.Uint64() & 1) == 1
}

func GenUint64(rng *rand.Rand) uint64 {
	return rng.Uint64()
}

func MakeGenUint64Range(min, max uint64) func(rng *rand.Rand) uint64 {
	return func(rng *rand.Rand) uint64 {
		return GenUint64Range(rng, min, max)
	}
}

func GenUint64Range(rng *rand.Rand, min, max uint64) uint64 {
	delta := max - min
	return min + (rng.Uint64() % delta)
}

func MakeGenUintRange(min, max uint) func(rng *rand.Rand) uint {
	return func(rng *rand.Rand) uint {
		return GenUintRange(rng, min, max)
	}
}

func GenUintRange(rng *rand.Rand, min, max uint) uint {
	return uint(GenUint64Range(rng, uint64(min), uint64(max)))
}
