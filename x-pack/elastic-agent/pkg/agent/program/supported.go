// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Code generated by x-pack/dev-tools/cmd/buildspec/buildspec.go - DO NOT EDIT.

package program

import (
	"strings"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/packer"
)

var Supported []Spec
var SupportedMap map[string]Spec

func init() {
	// Packed Files
	// spec/apm-server.yml
	// spec/endpoint.yml
	// spec/filebeat.yml
	// spec/fleet-server.yml
	// spec/heartbeat.yml
	// spec/metricbeat.yml
	// spec/osquerybeat.yml
	// spec/packetbeat.yml
	unpacked := packer.MustUnpack("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")
	SupportedMap = make(map[string]Spec)

	for f, v := range unpacked {
		s, err := NewSpecFromBytes(v)
		if err != nil {
			panic("Cannot read spec from " + f)
		}
		Supported = append(Supported, s)
		SupportedMap[strings.ToLower(s.Cmd)] = s
	}
}
