// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package mage

import (
	"sort"
	"strings"

	"github.com/pkg/errors"
)

// BuildPlatforms is a list of GOOS/GOARCH pairs supported by Go.
// The list originated from 'go tool dist list -json'.
var BuildPlatforms = BuildPlatformList{
	{"android/386", CGOSupported},
	{"android/amd64", CGOSupported},
	{"android/arm", CGOSupported},
	{"android/arm64", CGOSupported},
	{"darwin/386", CGOSupported | CrossBuildSupported},
	{"darwin/amd64", CGOSupported | CrossBuildSupported | Default},
	{"darwin/arm", CGOSupported},
	{"darwin/arm64", CGOSupported},
	{"dragonfly/amd64", CGOSupported},
	{"freebsd/386", CGOSupported},
	{"freebsd/amd64", CGOSupported},
	{"freebsd/arm", 0},
	{"linux/386", CGOSupported | CrossBuildSupported | Default},
	{"linux/amd64", CGOSupported | CrossBuildSupported | Default},
	{"linux/armv5", CGOSupported | CrossBuildSupported},
	{"linux/armv6", CGOSupported | CrossBuildSupported},
	{"linux/armv7", CGOSupported | CrossBuildSupported},
	{"linux/arm64", CGOSupported | CrossBuildSupported | Default},
	{"linux/mips", CGOSupported | CrossBuildSupported},
	{"linux/mips64", CGOSupported | CrossBuildSupported},
	{"linux/mips64le", CGOSupported | CrossBuildSupported},
	{"linux/mipsle", CGOSupported | CrossBuildSupported},
	{"linux/ppc64", CrossBuildSupported},
	{"linux/ppc64le", CGOSupported | CrossBuildSupported},
	{"linux/s390x", CGOSupported | CrossBuildSupported},
	{"nacl/386", 0},
	{"nacl/amd64p32", 0},
	{"nacl/arm", 0},
	{"netbsd/386", CGOSupported},
	{"netbsd/amd64", CGOSupported},
	{"netbsd/arm", CGOSupported},
	{"openbsd/386", CGOSupported},
	{"openbsd/amd64", CGOSupported},
	{"openbsd/arm", 0},
	{"plan9/386", 0},
	{"plan9/amd64", 0},
	{"plan9/arm", 0},
	{"solaris/amd64", CGOSupported},
	{"windows/386", CGOSupported | CrossBuildSupported | Default},
	{"windows/amd64", CGOSupported | CrossBuildSupported | Default},
}

// PlatformFeature specifies features that are supported for a platform.
type PlatformFeature uint8

// List of PlatformFeature types.
const (
	CGOSupported        PlatformFeature = 1 << iota // CGO is supported.
	CrossBuildSupported                             // Cross-build supported by golang-crossbuild.
	Default                                         // Built by default on crossBuild and package.
)

var platformFlagNames = map[PlatformFeature]string{
	CGOSupported:        "cgo",
	CrossBuildSupported: "xbuild",
	Default:             "default",
}

// String returns a string representation of the platform features.
func (f PlatformFeature) String() string {
	if f == 0 {
		return "none"
	}

	var names []string
	for value, name := range platformFlagNames {
		if f&value > 0 {
			names = append(names, name)
		}
	}

	return strings.Join(names, "|")
}

// CanCrossBuild returns true if cross-building is supported by
// golang-crossbuild.
func (f PlatformFeature) CanCrossBuild() bool {
	return f&CrossBuildSupported > 0
}

// SupportsCGO returns true if CGO is supported.
func (f PlatformFeature) SupportsCGO() bool {
	return f&CGOSupported > 0
}

// BuildPlatform represents a target platform for builds.
type BuildPlatform struct {
	Name  string
	Flags PlatformFeature
}

// GOOS returns the GOOS value contained in the name.
func (p BuildPlatform) GOOS() string {
	idx := strings.IndexByte(p.Name, '/')
	if idx == -1 {
		return p.Name
	}
	return p.Name[:idx]
}

// Arch returns the architecture value contained in the name.
func (p BuildPlatform) Arch() string {
	idx := strings.IndexByte(p.Name, '/')
	if idx == -1 {
		return ""
	}
	return p.Name[strings.IndexByte(p.Name, '/')+1:]
}

// GOARCH returns the GOARCH value associated with the architecture contained
// in the name. For ARM the Arch and GOARCH can differ because the GOARM value
// is encoded in the Arch value.
func (p BuildPlatform) GOARCH() string {
	// Allow armv7 to be interpreted as GOARCH=arm GOARM=7.
	arch := p.Arch()
	if strings.HasPrefix(arch, "armv") {
		return "arm"
	}
	return arch
}

// GOARM returns the ARM version.
func (p BuildPlatform) GOARM() string {
	arch := p.Arch()
	if strings.HasPrefix(arch, "armv") {
		return strings.TrimPrefix(arch, "armv")
	}
	return ""
}

// Attributes returns a new PlatformAttributes.
func (p BuildPlatform) Attributes() PlatformAttributes {
	return MakePlatformAttributes(p.GOOS(), p.GOARCH(), p.GOARM())
}

// PlatformAttributes contains all of the data that can be extracted from a
// BuildPlatform name.
type PlatformAttributes struct {
	Name   string
	GOOS   string
	GOARCH string
	GOARM  string
	Arch   string
}

// MakePlatformAttributes returns a new PlatformAttributes.
func MakePlatformAttributes(goos, goarch, goarm string) PlatformAttributes {
	arch := goarch
	if goarch == "arm" && goarm != "" {
		arch += "v" + goarm
	}

	name := goos
	if arch != "" {
		name += "/" + arch
	}

	return PlatformAttributes{
		Name:   name,
		GOOS:   goos,
		GOARCH: goarch,
		GOARM:  goarm,
		Arch:   arch,
	}
}

// String returns the string representation of the platform which has the format
// of "GOOS/Arch".
func (p PlatformAttributes) String() string {
	return p.Name
}

// BuildPlatformList is a list of BuildPlatforms that supports filtering.
type BuildPlatformList []BuildPlatform

// Returns all BuildPlatform names
func (list BuildPlatformList) Names() []string {
	platforms := make([]string, len(list))
	for i, bp := range list {
		platforms[i] = bp.Name
	}
	return platforms
}

// Get returns the BuildPlatform matching the given name.
func (list BuildPlatformList) Get(name string) (BuildPlatform, bool) {
	for _, bp := range list {
		if bp.Name == name {
			return bp, true
		}
	}
	return BuildPlatform{}, false
}

// Defaults returns the default platforms contained in the list.
func (list BuildPlatformList) Defaults() BuildPlatformList {
	return list.filter(func(p BuildPlatform) bool {
		return p.Flags&Default > 0
	})
}

// CrossBuild returns the platforms that support cross-building.
func (list BuildPlatformList) CrossBuild() BuildPlatformList {
	return list.filter(func(p BuildPlatform) bool {
		return p.Flags&CrossBuildSupported > 0
	})
}

// filter returns the platforms that match the given predicate.
func (list BuildPlatformList) filter(pred func(p BuildPlatform) bool) BuildPlatformList {
	var out BuildPlatformList
	for _, item := range list {
		if pred(item) {
			out = append(out, item)
		}
	}
	return out
}

// Remove returns a copy of list without platforms matching name.
func (list BuildPlatformList) Remove(name string) BuildPlatformList {
	attrs := BuildPlatform{Name: name}.Attributes()

	if attrs.Arch == "" {
		// Filter by GOOS only.
		return list.filter(func(bp BuildPlatform) bool {
			return bp.GOOS() != attrs.GOOS
		})
	}

	return list.filter(func(bp BuildPlatform) bool {
		return !(bp.GOOS() == attrs.GOOS && bp.Arch() == attrs.Arch)
	})
}

// Select returns a new list containing the platforms that match name.
func (list BuildPlatformList) Select(name string) BuildPlatformList {
	attrs := BuildPlatform{Name: name}.Attributes()

	if attrs.Arch == "" {
		// Filter by GOOS only.
		return list.filter(func(bp BuildPlatform) bool {
			return bp.GOOS() == attrs.GOOS
		})
	}

	return list.filter(func(bp BuildPlatform) bool {
		return bp.GOOS() == attrs.GOOS && bp.Arch() == attrs.Arch
	})
}

type platformExpression struct {
	Add              []string
	Select           []string
	SelectCrossBuild bool
	Remove           []string
}

func newPlatformExpression(expr string) (*platformExpression, error) {
	if strings.TrimSpace(expr) == "" {
		return nil, nil
	}

	pe := &platformExpression{}

	// Parse the expression.
	words := strings.FieldsFunc(expr, isSeparator)
	for _, w := range words {
		if strings.HasPrefix(w, "+") {
			pe.Add = append(pe.Add, strings.TrimPrefix(w, "+"))
		} else if strings.HasPrefix(w, "!") {
			pe.Remove = append(pe.Remove, strings.TrimPrefix(w, "!"))
		} else if w == "xbuild" {
			pe.SelectCrossBuild = true
		} else {
			pe.Select = append(pe.Select, w)
		}
	}

	// Validate the names used.
	checks := make([]string, 0, len(pe.Add)+len(pe.Select)+len(pe.Remove))
	checks = append(checks, pe.Add...)
	checks = append(checks, pe.Select...)
	checks = append(checks, pe.Remove...)

	for _, name := range checks {
		if name == "all" || name == "defaults" {
			continue
		}

		var valid bool
		for _, bp := range BuildPlatforms {
			if bp.Name == name || bp.GOOS() == name {
				valid = true
				break
			}
		}

		if !valid {
			return nil, errors.Errorf("invalid platform in expression: %v", name)
		}
	}

	return pe, nil
}

// NewPlatformList returns a new BuildPlatformList based on given expression.
//
// By default the initial set include only the platforms designated as defaults.
// To add additional platforms to list use an addition term that is designated
// with a plug sign (e.g. "+netbsd" or "+linux/armv7"). Or you may use "+all"
// to change the initial set to include all possible platforms then filter
// from there (e.g. "+all linux windows").
//
// The expression can consists of selections (e.g. "linux") and/or
// removals (e.g."!windows"). Each term can be valid GOOS or a valid GOOS/Arch
// pair.
//
// "xbuild" is a special selection term used to select all platforms that are
// cross-build eligible.
// "defaults" is a special selection or removal term that contains all platforms
// designated as a default.
// "all" is a special addition term for adding all valid GOOS/Arch pairs to the
// set.
func NewPlatformList(expr string) BuildPlatformList {
	pe, err := newPlatformExpression(expr)
	if err != nil {
		panic(err)
	}
	if pe == nil {
		return BuildPlatforms.Defaults()
	}

	var out BuildPlatformList
	if len(pe.Add) == 0 || (len(pe.Select) == 0 && len(pe.Remove) == 0) {
		// Bootstrap list with default platforms when the expression is
		// exclusively adds OR exclusively selects and removes.
		out = BuildPlatforms.Defaults()
	}

	all := BuildPlatforms
	for _, name := range pe.Add {
		if name == "all" {
			out = make(BuildPlatformList, len(all))
			copy(out, all)
			break
		}
		out = append(out, all.Select(name)...)
	}

	if len(pe.Select) > 0 {
		var selected BuildPlatformList
		for _, name := range pe.Select {
			selected = append(selected, out.Select(name)...)
		}
		out = selected
	}

	for _, name := range pe.Remove {
		if name == "defaults" {
			for _, defaultBP := range all.Defaults() {
				out = out.Remove(defaultBP.Name)
			}
			continue
		}
		out = out.Remove(name)
	}

	if pe.SelectCrossBuild {
		out = out.CrossBuild()
	}
	return out.deduplicate()
}

// Filter creates a new list based on the provided expression.
//
// The expression can consists of selections (e.g. "linux") and/or
// removals (e.g."!windows"). Each term can be valid GOOS or a valid GOOS/Arch
// pair.
//
// "xbuild" is a special selection term used to select all platforms that are
// cross-build eligible.
// "defaults" is a special selection or removal term that contains all platforms
// designated as a default.
func (list BuildPlatformList) Filter(expr string) BuildPlatformList {
	pe, err := newPlatformExpression(expr)
	if err != nil {
		panic(err)
	}
	if pe == nil {
		return list
	}
	if len(pe.Add) > 0 {
		panic(errors.Errorf("adds (%v) cannot be used in filter expressions",
			strings.Join(pe.Add, ", ")))
	}

	var out BuildPlatformList
	if len(pe.Select) == 0 && !pe.SelectCrossBuild {
		// Filter is only removals so clone the original list.
		out = append(out, list...)
	}

	if pe.SelectCrossBuild {
		out = append(out, list.CrossBuild()...)
	}
	for _, name := range pe.Select {
		if name == "defaults" {
			out = append(out, list.Defaults()...)
			continue
		}
		out = append(out, list.Select(name)...)
	}

	for _, name := range pe.Remove {
		if name == "defaults" {
			for _, defaultBP := range BuildPlatforms.Defaults() {
				out = out.Remove(defaultBP.Name)
			}
			continue
		}
		out = out.Remove(name)
	}

	return out.deduplicate()
}

// Merge creates a new list with the two list merged.
func (list BuildPlatformList) Merge(with BuildPlatformList) BuildPlatformList {
	out := make(BuildPlatformList, 0, len(list)+len(with))
	out = append(list, with...)
	out = append(out, with...)
	return out.deduplicate()
}

// deduplicate removes duplicate platforms and sorts the list.
func (list BuildPlatformList) deduplicate() BuildPlatformList {
	set := map[string]BuildPlatform{}
	for _, item := range list {
		set[item.Name] = item
	}

	var out BuildPlatformList
	for _, v := range set {
		out = append(out, v)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}
