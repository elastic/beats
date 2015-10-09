package main

// This file is mandatory as otherwise the packetbeat.test binary is not generated correctly.

import (
	"flag"
	"testing"
)

var systemTest *bool

func init() {
	systemTest = flag.Bool("systemTest", false, "Set to true when running system tests")
}

// Test started when the test binary is started. Only calls main.
func TestSystem(t *testing.T) {

	if *systemTest {
		main()
	}
}
