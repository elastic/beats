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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildPlatform(t *testing.T) {
	bp := BuildPlatform{"windows/amd64", 0}
	assert.Equal(t, "windows", bp.GOOS())
	assert.Equal(t, "amd64", bp.GOARCH())
	assert.Equal(t, "", bp.GOARM())
	assert.Equal(t, "amd64", bp.Arch())

	bp = BuildPlatform{"linux/armv7", 0}
	assert.Equal(t, "linux", bp.GOOS())
	assert.Equal(t, "arm", bp.GOARCH())
	assert.Equal(t, "7", bp.GOARM())
	assert.Equal(t, "armv7", bp.Arch())
	attrs := bp.Attributes()
	assert.Equal(t, bp.Name, attrs.Name)
	assert.Equal(t, "linux", attrs.GOOS)
	assert.Equal(t, "arm", attrs.GOARCH)
	assert.Equal(t, "7", attrs.GOARM)
	assert.Equal(t, "armv7", attrs.Arch)

	bp = BuildPlatform{"linux", 0}
	assert.Equal(t, "linux", bp.GOOS())
	assert.Equal(t, "", bp.GOARCH())
	assert.Equal(t, "", bp.GOARM())
	assert.Equal(t, "", bp.Arch())
	attrs = bp.Attributes()
	assert.Equal(t, bp.Name, attrs.Name)
	assert.Equal(t, "linux", attrs.GOOS)
	assert.Equal(t, "", attrs.GOARCH)
	assert.Equal(t, "", attrs.GOARM)
	assert.Equal(t, "", attrs.Arch)
}

func TestBuildPlatformsListRemove(t *testing.T) {
	list := BuildPlatformList{
		{"linux/amd64", 0},
		{"linux/386", 0},
	}

	assert.ElementsMatch(t,
		list.Remove("linux/386"),
		BuildPlatformList{{"linux/amd64", 0}},
	)
}

func TestBuildPlatformsListRemoveOS(t *testing.T) {
	list := BuildPlatformList{
		{"linux/amd64", 0},
		{"linux/386", 0},
		{"windows/amd64", 0},
	}

	assert.ElementsMatch(t,
		list.Remove("linux"),
		BuildPlatformList{{"windows/amd64", 0}},
	)
}

func TestBuildPlatformsListSelect(t *testing.T) {
	list := BuildPlatformList{
		{"linux/amd64", 0},
		{"linux/386", 0},
	}

	assert.ElementsMatch(t,
		list.Select("linux/386"),
		BuildPlatformList{{"linux/386", 0}},
	)
}

func TestBuildPlatformsListDefaults(t *testing.T) {
	list := BuildPlatformList{
		{"linux/amd64", Default},
		{"linux/386", 0},
	}

	assert.ElementsMatch(t,
		list.Defaults(),
		BuildPlatformList{{"linux/amd64", Default}},
	)
}

func TestBuildPlatformsListFilter(t *testing.T) {
	assert.Len(t, BuildPlatforms.Filter("!linux/armv7"), len(BuildPlatforms)-1)

	assert.Len(t, BuildPlatforms.Filter("solaris"), 1)
	assert.Len(t, BuildPlatforms.Defaults().Filter("solaris"), 0)

	assert.Len(t, BuildPlatforms.Filter("windows"), 2)
	assert.Len(t, BuildPlatforms.Filter("windows/386"), 1)
	assert.Len(t, BuildPlatforms.Filter("!defaults"), len(BuildPlatforms)-len(BuildPlatforms.Defaults()))

	defaults := BuildPlatforms.Defaults()
	assert.ElementsMatch(t,
		defaults.Filter("darwin"),
		defaults.Filter("!windows !linux"))
	assert.ElementsMatch(t,
		defaults,
		defaults.Filter("windows linux darwin"))
	assert.ElementsMatch(t,
		defaults,
		append(defaults.Filter("darwin"), defaults.Filter("!darwin")...))
	assert.ElementsMatch(t,
		BuildPlatforms,
		BuildPlatforms.Filter(""))
	assert.ElementsMatch(t,
		BuildPlatforms.Filter("defaults"),
		BuildPlatforms.Defaults())
}

func TestNewPlatformList(t *testing.T) {
	assert.Len(t, NewPlatformList("+all !linux/armv7"), len(BuildPlatforms)-1)
	assert.Len(t, NewPlatformList("+solaris"), len(BuildPlatforms.Defaults())+1)
	assert.Len(t, NewPlatformList("solaris"), 0)
	assert.Len(t, NewPlatformList("+all solaris"), 1)
	assert.Len(t, NewPlatformList("+windows"), len(BuildPlatforms.Defaults()))
	assert.Len(t, NewPlatformList("+linux/ppc64 !defaults"), 1)

	assert.ElementsMatch(t,
		NewPlatformList("darwin"),
		NewPlatformList("!windows !linux"))
	assert.ElementsMatch(t,
		BuildPlatforms.Defaults(),
		NewPlatformList("windows linux darwin"))
	assert.ElementsMatch(t,
		BuildPlatforms.Defaults(),
		append(NewPlatformList("darwin"), NewPlatformList("!darwin")...))
	assert.ElementsMatch(t,
		BuildPlatforms.Defaults(),
		NewPlatformList(""))
	assert.ElementsMatch(t,
		BuildPlatforms,
		NewPlatformList("+all"))
}
