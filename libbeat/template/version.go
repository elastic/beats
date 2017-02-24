package template

import (
	"fmt"
	"strconv"
	"strings"
)

type Version struct {
	version string
	major   int
	minor   int
	bugfix  int
	meta    string
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
		v.meta = tmp[1]
	}

	versions := strings.Split(version, ".")
	if len(versions) != 3 {
		return nil, fmt.Errorf("Passed version is not semver: %s", version)
	}

	var err error
	v.major, err = strconv.Atoi(versions[0])
	if err != nil {
		return nil, fmt.Errorf("Could not convert major to integer: %s", versions[0])
	}

	v.minor, err = strconv.Atoi(versions[1])
	if err != nil {
		return nil, fmt.Errorf("Could not convert minor to integer: %s", versions[1])
	}

	v.bugfix, err = strconv.Atoi(versions[2])
	if err != nil {
		return nil, fmt.Errorf("Could not convert bugfix to integer: %s", versions[2])
	}

	return &v, nil
}

func (v *Version) IsMajor(major int) bool {
	return major == v.major
}

func (v *Version) String() string {
	return v.version
}
