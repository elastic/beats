// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build (linux && 386) || (linux && amd64)
// +build linux,386 linux,amd64

package socket

import (
	"fmt"
	"strings"
	"testing"

	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/guess"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/helper"
	"github.com/elastic/beats/v7/x-pack/auditbeat/tracing"
)

func probeName(p tracing.Probe) string {
	return p.Name
}

func probeGroup(p tracing.Probe) string {
	return p.EffectiveGroup()
}

func probeArgs(p tracing.Probe) string {
	return p.Fetchargs
}

func validateProbe(p tracing.Probe) error {
	for _, check := range []struct {
		getter  func(p tracing.Probe) string
		pattern string
		reason  string
	}{
		{probeName, "-", "name has a dash"},
		{probeGroup, "-", "group has a dash"},
		{probeArgs, ":x", "fetchargs uses xNN type"},
		{probeArgs, ":string", "fetchargs uses string type"},
		{probeArgs, "$comm", "fetchargs uses $comm"},
	} {
		if strings.Contains(check.getter(p), check.pattern) {
			return fmt.Errorf("incompatible kprobe definition (%s): '%s'", check.reason, p.String())
		}
	}
	return nil
}

// These tests are to make KProbes that are not compatible with older kernels
// are not inadvertently introduced.
func validateProbeList(t *testing.T, probes []helper.ProbeDef) {
	for _, probe := range probes {
		t.Run(probe.Probe.Name, func(t *testing.T) {
			if err := validateProbe(probe.Probe); err != nil {
				t.Log(err)
				t.Fail()
			}
		})
	}
}

func TestRuntimeKProbesAreBackwardsCompatible(t *testing.T) {
	validateProbeList(t, getAllKProbes())
}

func TestGuessKProbesAreBackwardsCompatible(t *testing.T) {
	var probes []helper.ProbeDef
	for _, iface := range guess.Registry.GetList() {
		switch v := iface.(type) {
		case guess.Guesser:
			p, err := v.Probes()
			if err != nil {
				t.Fatal("error getting probes from", v.Name(), err)
			}
			probes = append(probes, p...)
		default:
			t.Fatalf("bad guess (type %T): %+v", v, v)
		}
	}
	validateProbeList(t, probes)
}
