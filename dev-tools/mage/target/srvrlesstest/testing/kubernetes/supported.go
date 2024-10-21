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

package kubernetes

import (
	"errors"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/define"
)

// ErrUnknownDockerVariant is the error returned when the variant is unknown.
var ErrUnknownDockerVariant = errors.New("unknown docker variant type")

// arches defines the list of supported architectures of Kubernetes
var arches = []string{define.AMD64, define.ARM64}

// versions defines the list of supported version of Kubernetes.
var versions = []define.OS{
	// Kubernetes 1.31
	{
		Type:    define.Kubernetes,
		Version: "1.31.0",
	},
	// Kubernetes 1.30
	{
		Type:    define.Kubernetes,
		Version: "1.30.2",
	},
	// Kubernetes 1.29
	{
		Type:    define.Kubernetes,
		Version: "1.29.4",
	},
	// Kubernetes 1.28
	{
		Type:    define.Kubernetes,
		Version: "1.28.9",
	},
}

// variants defines the list of variants and the image name for that variant.
//
// Note: This cannot be a simple map as the order matters. We need the
// one that we want to be the default test to be first.
var variants = []struct {
	Name  string
	Image string
}{
	{
		Name:  "basic",
		Image: "docker.elastic.co/beats/elastic-agent",
	},
	{
		Name:  "ubi",
		Image: "docker.elastic.co/beats/elastic-agent-ubi",
	},
	{
		Name:  "wolfi",
		Image: "docker.elastic.co/beats/elastic-agent-wolfi",
	},
	{
		Name:  "complete",
		Image: "docker.elastic.co/beats/elastic-agent-complete",
	},
	{
		Name:  "complete-wolfi",
		Image: "docker.elastic.co/beats/elastic-agent-complete-wolfi",
	},
	{
		Name:  "cloud",
		Image: "docker.elastic.co/beats-ci/elastic-agent-cloud",
	},
	{
		Name:  "service",
		Image: "docker.elastic.co/beats-ci/elastic-agent-service",
	},
}

// GetSupported returns the list of supported OS types for Kubernetes.
func GetSupported() []define.OS {
	supported := make([]define.OS, 0, len(versions)*len(variants)*2)
	for _, a := range arches {
		for _, v := range versions {
			for _, variant := range variants {
				c := v
				c.Arch = a
				c.DockerVariant = variant.Name
				supported = append(supported, c)
			}
		}
	}
	return supported
}

// VariantToImage returns the image name from the variant.
func VariantToImage(variant string) (string, error) {
	for _, v := range variants {
		if v.Name == variant {
			return v.Image, nil
		}
	}
	return "", ErrUnknownDockerVariant
}
