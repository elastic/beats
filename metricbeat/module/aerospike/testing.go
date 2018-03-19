package aerospike

import (
	"os"
)

// Helper functions for testing used in the aerospike MetricSets.

// GetAerospikeEnvHost returns the hostname of the Aerospike server to use for
// testing. It reads the value from the AEROSPIKE_HOST environment variable and
// returns localhost if it is not set.
func GetAerospikeEnvHost() string {
	host := os.Getenv("AEROSPIKE_HOST")

	if len(host) == 0 {
		host = "localhost"
	}
	return host
}

// GetAerospikeEnvPort returns the port of the Aerospike server to use for
// testing. It reads the value from the AEROSPIKE_PORT environment variable and
// returns 3000 if it is not set.
func GetAerospikeEnvPort() string {
	port := os.Getenv("AEROSPIKE_PORT")

	if len(port) == 0 {
		port = "3000"
	}
	return port
}
