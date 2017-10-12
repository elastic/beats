package common

import (
	"fmt"
	"strconv"
	"strings"
)

type Version struct {
	version string
	Major   int
	Minor   int
	Bugfix  int
	Meta    string
}

// NewVersion expects a string in the format:
// major.minor.bugfix(-meta)
func NewVersion(version string) (*Version, error) {
	v := Version{
		version: version,
	}

	// Check for meta info
	if strings.Contains(version, "-") {
		tmp := strings.Split(version, "-")
		version = tmp[0]
		v.Meta = tmp[1]
	}

	versions := strings.Split(version, ".")
	if len(versions) != 3 {
		return nil, fmt.Errorf("Passed version is not semver: %s", version)
	}

	var err error
	v.Major, err = strconv.Atoi(versions[0])
	if err != nil {
		return nil, fmt.Errorf("Could not convert major to integer: %s", versions[0])
	}

	v.Minor, err = strconv.Atoi(versions[1])
	if err != nil {
		return nil, fmt.Errorf("Could not convert minor to integer: %s", versions[1])
	}

	v.Bugfix, err = strconv.Atoi(versions[2])
	if err != nil {
		return nil, fmt.Errorf("Could not convert bugfix to integer: %s", versions[2])
	}

	return &v, nil
}

func (v *Version) IsMajor(major int) bool {
	return major == v.Major
}

// LessThan returns true if v is strictly smaller than v1. When comparing, the major,
// minor, bugfix and pre-release numbers are compared in order.
func (v *Version) LessThanOrEqual(withMeta bool, v1 *Version) bool {
	if withMeta && v.version == v1.version {
		return true
	}
	if v.Major < v1.Major {
		return true
	}
	if v.Major == v1.Major {
		if v.Minor < v1.Minor {
			return true
		}
		if v.Minor == v1.Minor {
			if v.Bugfix < v1.Bugfix {
				return true
			}
			if v.Bugfix == v1.Bugfix {
				if withMeta {
					return v.metaIsLessThanOrEqual(v1)
				} else {
					return true
				}
			}
		}
	}
	return false
}

// LessThan returns true if v is strictly smaller than v1. When comparing, the major,
// minor and bugfix numbers are compared in order. The meta part is not taken into account.
func (v *Version) LessThan(v1 *Version) bool {
	if v.Major < v1.Major {
		return true
	} else if v.Major == v1.Major {
		if v.Minor < v1.Minor {
			return true
		} else if v.Minor == v1.Minor {
			if v.Bugfix < v1.Bugfix {
				return true
			}
		}
	}
	return false
}

func (v *Version) String() string {
	return v.version
}

func (v *Version) metaIsLessThanOrEqual(v1 *Version) bool {
	if v.Meta == "" && v1.Meta == "" {
		return true
	}
	if v.Meta == "" {
		return false
	}
	if v1.Meta == "" {
		return true
	}
	return v.Meta <= v1.Meta
}
