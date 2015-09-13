package main

// This file is mandatory as otherwise the packetbeat.test binary is not generated correctly. Reason???

import (
	"testing"
	"flag"
)

var integration *bool

func init() {
	integration = flag.Bool("integration", false, "Set to true when running integration tests")
}

// Test started when the test binary is started. Only calls main.
func TestIntegration(t *testing.T) {

	if *integration {
		main()
	}
}
