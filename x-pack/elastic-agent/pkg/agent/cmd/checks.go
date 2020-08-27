// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !windows

package cmd

import (
	"github.com/spf13/cobra"

	// import logp flags
	_ "github.com/elastic/beats/v7/libbeat/logp/configure"
)

// preRunCheck is noop because
// - darwin.tar - symlink created during packaging
// - linux.tar - symlink created during packaging
// - linux.rpm - symlink created during packaging
// - linux.deb - symlink created during packaging
func preRunCheck(flags *globalFlags) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		return nil
	}
}
