package rpm

import (
	"fmt"
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
	PackageVersion

	// DepFlag constants
	Flags() int
}

// private basic implementation or a package dependency.
type dependency struct {
	flags   int
	name    string
	epoch   int
	version string
	release string
}

// Dependencies are a slice of Dependency interfaces.
type Dependencies []Dependency

// NewDependency returns a new instance of a package dependency definition.
func NewDependency(flgs int, name string, epoch int, version string, release string) Dependency {
	return &dependency{
		flags:   flgs,
		name:    name,
		epoch:   epoch,
		version: version,
		release: release,
	}
}

// String returns a string representation a package dependency in a similar
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
