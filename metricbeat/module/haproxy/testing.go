package haproxy

import "os"

// Helper functions for testing used in the haproxy metricsets

// GetEnvHost returns the hostname of the HAProxy server to use for testing.
// It reads the value from the HAPROXY_HOST environment variable and returns
// 127.0.0.1 if it is not set.
func GetEnvHost() string {
	host := os.Getenv("HAPROXY_HOST")

	if len(host) == 0 {
		host = "127.0.0.1"
	}
	return host
}

// GetRedisEnvPort returns the port of the HAProxy server to use for testing.
// It reads the value from the HAPROXY_PORT environment variable and returns
// 14567 if it is not set.
func GetEnvPort() string {
	port := os.Getenv("HAPROXY_PORT")

	if len(port) == 0 {
		port = "14567"
	}
	return port
}
