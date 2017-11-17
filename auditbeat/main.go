package main

import (
	"os"

	"github.com/elastic/beats/auditbeat/cmd"

	_ "github.com/elastic/beats/auditbeat/module/audit"
	_ "github.com/elastic/beats/auditbeat/module/audit/file"
	_ "github.com/elastic/beats/auditbeat/module/audit/kernel"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
