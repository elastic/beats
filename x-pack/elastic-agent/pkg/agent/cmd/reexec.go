// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows
// +build !windows

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
)

<<<<<<< HEAD:x-pack/elastic-agent/pkg/agent/cmd/reexec.go
func newReExecWindowsCommand(_ []string, streams *cli.IOStreams) *cobra.Command {
	return nil
=======
// pipelineFS holds the yml representation of the ingest node pipelines
//
//go:embed */ingest/*.yml
var pipelinesFS embed.FS

func init() {
	module.PipelinesFS = &pipelinesFS
>>>>>>> e7e6dacfca ([updatecli][githubrelease] Bump version to 1.19.5 (#34497)):x-pack/winlogbeat/module/pipeline.go
}
