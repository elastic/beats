/*
Package version provides functions for comparing RPM package versions.
This implementation is tested for correctness by passing the same inputs to the
original implementation.
*/
package version

import (
	"math"
	"regexp"
	"strings"
	"unicode"
)

// alphanumPattern is a regular expression to match all sequences of numeric
// characters or alphanumeric characters.
var alphanumPattern = regexp.MustCompile("([a-zA-Z]+)|([0-9]+)|(~)")

// Interface is an interface which holds version information for a single
// package.
type Interface interface {
	Name() string
	Epoch() int
	Version() string
	Release() string
}

// Compare compares the version details of two packages. Versions are
// compared by Epoch, Version and Release in descending order of precedence.
//
// If a is more recent than b, 1 is returned. If a is less recent than b, -1 is
// returned. If a and b are equal, 0 is returned.
//
// This function does not consider if the two packages have the same name or if
// either package has been made obsolete by the other.
func Compare(a, b Interface) int {
	// compare nils
	if a == nil && b == nil {
		return 0
	} else if a == nil {
		return -1
	} else if b == nil {
		return 1
	}

	// compare epoch
	ae := a.Epoch()
	be := b.Epoch()
	if ae != be {
		if ae > be {
			return 1
		}
		return -1
	}

	// compare version
	if rc := rpmvercmp(a.Version(), b.Version()); rc != 0 {
		return rc
	}

	// compare release
	return rpmvercmp(a.Release(), b.Release())
}

// rpmcmpver compares two version or release strings.
//
// For the original C implementation, see:
// https://github.com/rpm-software-management/rpm/blob/master/lib/rpmvercmp.c#L16
func rpmvercmp(a, b string) int {
	// shortcut for equality
	if a == b {
		return 0
	}

	// get alpha/numeric segements
	segsa := alphanumPattern.FindAllString(a, -1)
	segsb := alphanumPattern.FindAllString(b, -1)
	segs := int(math.Min(float64(len(segsa)), float64(len(segsb))))

	// compare each segment
	for i := 0; i < segs; i++ {
		a := segsa[i]
		b := segsb[i]

		// compare tildes
		if []rune(a)[0] == '~' || []rune(b)[0] == '~' {
			if []rune(a)[0] != '~' {
				return 1
			}
			if []rune(b)[0] != '~' {
				return -1
			}
		}

		if unicode.IsNumber([]rune(a)[0]) {
			// numbers are always greater than alphas
			if !unicode.IsNumber([]rune(b)[0]) {
				// a is numeric, b is alpha
				return 1
			}

			// trim leading zeros
			a = strings.TrimLeft(a, "0")
			b = strings.TrimLeft(b, "0")

			// longest string wins without further comparison
			if len(a) > len(b) {
				return 1
			} else if len(b) > len(a) {
				return -1
			}

		} else if unicode.IsNumber([]rune(b)[0]) {
			// a is alpha, b is numeric
			return -1
		}

		// string compare
		if a < b {
			return -1
		} else if a > b {
			return 1
		}
	}

	// segments were all the same but separators must have been different
	if len(segsa) == len(segsb) {
		return 0
	}

	// If there is a tilde in a segment past the min number of segments, find it.
	if len(segsa) > segs && []rune(segsa[segs])[0] == '~' {
		return -1
	} else if len(segsb) > segs && []rune(segsb[segs])[0] == '~' {
		return 1
	}

	// whoever has the most segments wins
	if len(segsa) > len(segsb) {
		return 1
	}
	return -1
}
