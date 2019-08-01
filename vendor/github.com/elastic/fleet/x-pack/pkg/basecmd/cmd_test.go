// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package basecmd

import (
	"testing"

	"github.com/elastic/fleet/x-pack/pkg/cli"
)

func TestBaseCmd(t *testing.T) {
	streams, _, _, _ := cli.NewTestingIOStreams()
	NewDefaultCommandsWithArgs([]string{}, streams)
}
