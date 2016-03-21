/*

Helper functions for testing used in the redis metricsets

*/
package redis

import (
	"os"
)

func GetRedisEnvHost() string {
	host := os.Getenv("REDIS_HOST")

	if len(host) == 0 {
		host = "127.0.0.1"
	}
	return host
}

func GetRedisEnvPort() string {
	port := os.Getenv("REDIS_PORT")

	if len(port) == 0 {
		port = "6379"
	}
	return port
}
