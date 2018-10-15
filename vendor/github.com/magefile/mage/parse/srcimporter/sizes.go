// +build !go1.9

package srcimporter

import "go/types"

// common architecture word sizes and alignments
var gcArchSizes = map[string]*types.StdSizes{
	"386":      {4, 4},
	"arm":      {4, 4},
	"arm64":    {8, 8},
	"amd64":    {8, 8},
	"amd64p32": {4, 8},
	"mips":     {4, 4},
	"mipsle":   {4, 4},
	"mips64":   {8, 8},
	"mips64le": {8, 8},
	"ppc64":    {8, 8},
	"ppc64le":  {8, 8},
	"s390x":    {8, 8},
	// When adding more architectures here,
	// update the doc string of SizesFor below.
}

// SizesFor returns the Sizes used by a compiler for an architecture.
// The result is nil if a compiler/architecture pair is not known.
//
// Supported architectures for compiler "gc":
// "386", "arm", "arm64", "amd64", "amd64p32", "mips", "mipsle",
// "mips64", "mips64le", "ppc64", "ppc64le", "s390x".
func SizesFor(compiler, arch string) types.Sizes {
	if compiler != "gc" {
		return nil
	}
	s, ok := gcArchSizes[arch]
	if !ok {
		return nil
	}
	return s
}
