package main

// This file is mandatory as otherwise the packetbeat.test binary is not generated correctly. Reason???

import (
	"testing"
)

// Test started when the test binary is started. Only calls main.
func TestIntegration(t *testing.T) {
	main()
}
