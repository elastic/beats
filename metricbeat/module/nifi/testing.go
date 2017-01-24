package nifi

import "os"

// Helper functions for testing nifi metricsets.

// GetEnvHost returns the hostname of the nifi server to use for testing.
// It reads the value from the NIFI_HOST environment variable and returns
// localhost if it is not set.
func GetEnvHost() string {
	host := os.Getenv("NIFI_HOST")

	if len(host) == 0 {
		host = "localhost"
	}
	return host
}

// GetEnvPort returns the port of the nifi server to use for testing.
// It reads the value from the NIFI_PORT environment variable and returns
// 8080 if it is not set.
func GetEnvPort() string {
	port := os.Getenv("NIFI_PORT")

	if len(port) == 0 {
		port = "8080"
	}
	return port
}
