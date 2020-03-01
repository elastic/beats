package rpm

import (
	"fmt"

	"github.com/cavaliercoder/go-rpm/version"
)

// Dependency flags indicate how versions comparisons should be computed when
// comparing versions of dependent packages.
const (
	DepFlagAny            = 0
	DepFlagLesser         = (1 << 1)
	DepFlagGreater        = (1 << 2)
	DepFlagEqual          = (1 << 3)
	DepFlagLesserOrEqual  = (DepFlagEqual | DepFlagLesser)
	DepFlagGreaterOrEqual = (DepFlagEqual | DepFlagGreater)
)

// See: https://github.com/rpm-software-management/rpm/blob/master/lib/rpmds.h#L25

// Dependency is an interface which represents a relationship between two
// packages. It might indicate that one package requires, conflicts with,
// obsoletes or provides another package.
//
// Dependency implements the PackageVersion interface and so may be used when
// comparing versions with other types of packages.
type Dependency interface {
	version.Interface

	// DepFlag constants
	Flags() int
}

// private basic implementation of a package dependency.
type dependency struct {
	flags   int
	name    string
	epoch   int
	version string
	release string
}

// Flags determines the nature of the package relationship and the comparison
// used for the given version constraint.
func (c *dependency) Flags() int {
	return c.flags
}

// Name is the name of the package target package.
func (c *dependency) Name() string {
	return c.name
}

// Epoch is the epoch constraint of the target package.
func (c *dependency) Epoch() int {
	return c.epoch
}

// Version is the version constraint of the target package.
func (c *dependency) Version() string {
	return c.version
}

// Release is the release constraint of the target package.
func (c *dependency) Release() string {
	return c.release
}

// String returns a string representation of a package dependency in a similar
// format to `rpm -qR`.
func (c *dependency) String() string {
	s := c.name

	switch {
	case DepFlagLesserOrEqual == (c.flags & DepFlagLesserOrEqual):
		s = fmt.Sprintf("%s <=", s)

	case DepFlagLesser == (c.flags & DepFlagLesser):
		s = fmt.Sprintf("%s <", s)

	case DepFlagGreaterOrEqual == (c.flags & DepFlagGreaterOrEqual):
		s = fmt.Sprintf("%s >=", s)

	case DepFlagGreater == (c.flags & DepFlagGreater):
		s = fmt.Sprintf("%s >", s)

	case DepFlagEqual == (c.flags & DepFlagEqual):
		s = fmt.Sprintf("%s =", s)
	}

	if c.version != "" {
		s = fmt.Sprintf("%s %s", s, c.version)
	}

	if c.release != "" {
		s = fmt.Sprintf("%s.%s", s, c.release)
	}

	return s
}
