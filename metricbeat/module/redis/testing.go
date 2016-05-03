package redis

import (
	"os"
)

// Helper functions for testing used in the redis metricsets

// GetRedisEnvHost returns the hostname of the Redis server to use for testing.
// It reads the value from the REDIS_HOST environment variable and returns
// 127.0.0.1 if it is not set.
func GetRedisEnvHost() string {
	host := os.Getenv("REDIS_HOST")

	if len(host) == 0 {
		host = "127.0.0.1"
	}
	return host
}

// GetRedisEnvPort returns the port of the Redis server to use for testing.
// It reads the value from the REDIS_PORT environment variable and returns
// 6379 if it is not set.
func GetRedisEnvPort() string {
	port := os.Getenv("REDIS_PORT")

	if len(port) == 0 {
		port = "6379"
	}
	return port
}
