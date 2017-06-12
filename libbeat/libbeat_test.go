package main

import (
	"flag"
	"testing"
)

var systemTest *bool

func init() {
	systemTest = flag.Bool("systemTest", false, "Set to true when running system tests")

	RootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("systemTest"))
	RootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("test.coverprofile"))
}

// Test started when the test binary is started
func TestSystem(t *testing.T) {
	if *systemTest {
		main()
	}
}
