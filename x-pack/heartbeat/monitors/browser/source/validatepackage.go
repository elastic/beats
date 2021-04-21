// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package source

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/Masterminds/semver"
)

// ensure compatability of synthetics by enforcing the installed
// version never goes beyond this range
const ExpectedSynthVersion = "<2.0.0"

type packageJson struct {
	Dependencies struct {
		SynthVersion string `json:"@elastic/synthetics"`
	} `json:"dependencies"`
	DevDependencies struct {
		SynthVersion string `json:"@elastic/synthetics"`
	} `json:"devDependencies"`
}

var nonNumberRegex = regexp.MustCompile("\\D")

// parsed a given dep version by ignoring all range tags (^, = , >, <)
func parseVersion(version string) string {
	dotParts := strings.SplitN(version, ".", 4)

	parsed := []string{}
	for _, v := range dotParts[:3] {
		value := nonNumberRegex.ReplaceAllString(v, "")
		parsed = append(parsed, value)
	}
	return strings.Join(parsed, ".")
}

func validateVersion(expected string, current string) error {
	expectedRange, err := semver.NewConstraint(expected)
	if err != nil {
		return err
	}

	parsed := parseVersion(current)
	currentVersion, err := semver.NewVersion(parsed)
	if err != nil {
		return fmt.Errorf("error parsing @elastic/synthetics version: '%s' %w", currentVersion, err)
	}

	isValid := expectedRange.Check(currentVersion)
	if !isValid {
		return fmt.Errorf("parsed @elastic/synthetics version '%s' is not compatible", current)
	}
	return nil
}

func validatePackageJSON(path string) error {
	pkgData, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("could not read file '%s': %w", path, err)
	}
	pkgJson := packageJson{}
	err = json.Unmarshal(pkgData, &pkgJson)
	if err != nil {
		return fmt.Errorf("could not unmarshall @elastic/synthetics version: %w", err)
	}

	synthVersion := pkgJson.Dependencies.SynthVersion
	if synthVersion == "" {
		synthVersion = pkgJson.DevDependencies.SynthVersion
	}
	err = validateVersion(ExpectedSynthVersion, synthVersion)
	if err != nil {
		return fmt.Errorf("could not validate @elastic/synthetics version: '%s' %w", synthVersion, err)
	}
	return nil
}
