package apache

import (
	"os"
)

// Helper functions for testing the Apache module's MetricSets.

// GetApacheEnvHost returns the apache server hostname to use for testing. It
// reads the value from the APACHE_HOST environment variable and returns
// 127.0.0.1 if it is not set.
func GetApacheEnvHost() string {
	host := os.Getenv("APACHE_HOST")

	if len(host) == 0 {
		host = "127.0.0.1"
	}
	return host
}

// GetApacheEnvPort returns the port of the apache server to use for testing.
// It reads the value from the APACHE_PORT environment variable and returns
// 80 if it is not set.
func GetApacheEnvPort() string {
	port := os.Getenv("APACHE_PORT")

	if len(port) == 0 {
		port = "80"
	}
	return port
}
