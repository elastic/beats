// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"testing"
)

func TestAgent(t *testing.T) {
	// t.Run("test agent with subcommand", func(t *testing.T) {
	// 	streams, _, _, _ := cli.NewTestingIOStreams()
	// 	cmd := NewCommandWithArgs([]string{}, streams)
	// 	cmd.SetOutput(streams.Out)
	// 	cmd.Execute()
	// })

	// t.Run("test run subcommand", func(t *testing.T) {
	// 	streams, _, out, _ := cli.NewTestingIOStreams()
	// 	cmd := newRunCommandWithArgs(globalFlags{
	// 		PathConfigFile: filepath.Join("build", "agent.yml"),
	// 	}, []string{}, streams)
	// 	cmd.SetOutput(streams.Out)
	// 	cmd.Execute()
	// 	contents, err := ioutil.ReadAll(out)
	// 	if !assert.NoError(t, err) {
	// 		return
	// 	}
	// 	assert.True(t, strings.Contains(string(contents), "Hello I am running"))
	// })
}
