// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package version

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/fleet/x-pack/pkg/cli"
)

func TestCmd(t *testing.T) {
	streams, _, out, _ := cli.NewTestingIOStreams()
	NewCommandWithArgs(streams).Execute()
	version, err := ioutil.ReadAll(out)

	if !assert.NoError(t, err) {
		return
	}
	assert.True(t, strings.Contains(string(version), "Agent version is"))
}
