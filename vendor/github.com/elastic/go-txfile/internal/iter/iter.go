// Package iter provides functions for common array iteration strategies.
package iter

// Fn type for range based iterators.
type Fn func(len int) (begin, end int, next func(int) int)

// Forward returns limits and next function for forward iteration.
func Forward(l int) (begin, end int, next func(int) int) {
	return 0, l, func(i int) int { return i + 1 }
}

// Reversed returns limits and next function for reverse iteration.
func Reversed(l int) (begin, end int, next func(int) int) {
	return l - 1, -1, func(i int) int { return i - 1 }
}
