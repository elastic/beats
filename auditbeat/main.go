package main

import (
	"os"

	"github.com/elastic/beats/auditbeat/cmd"

	// Register modules.
	_ "github.com/elastic/beats/auditbeat/module/auditd"
	_ "github.com/elastic/beats/auditbeat/module/file_integrity"

	// Register includes.
	_ "github.com/elastic/beats/auditbeat/include"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
