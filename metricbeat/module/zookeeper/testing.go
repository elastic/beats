package zookeeper

import (
	"os"
)

// Helper functions for testing used in the zookeeper MetricSets.

// GetZookeeperEnvHost returns the hostname of the ZooKeeper server to use for
// testing. It reads the value from the ZOOKEEPER_HOST environment variable and
// returns localhost if it is not set.
func GetZookeeperEnvHost() string {
	host := os.Getenv("ZOOKEEPER_HOST")

	if len(host) == 0 {
		host = "localhost"
	}
	return host
}

// GetZookeeperEnvPort returns the port of the ZooKeeper server to use for
// testing. It reads the value from the ZOOKEEPER_PORT environment variable and
// returns 2181 if it is not set.
func GetZookeeperEnvPort() string {
	port := os.Getenv("ZOOKEEPER_PORT")

	if len(port) == 0 {
		port = "2181"
	}
	return port
}
