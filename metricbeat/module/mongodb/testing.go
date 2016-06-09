package mongodb

import "os"

// Helper functions for testing used in the mongodb metricsets

// GetEnvHost returns the hostname of the Mongodb server to use for testing.
// It reads the value from the MONGODB_HOST environment variable and returns
// 127.0.0.1 if it is not set.
func GetEnvHost() string {
	host := os.Getenv("MONGODB_HOST")

	if len(host) == 0 {
		host = "127.0.0.1"
	}
	return host
}

// GetMongodbEnvPort returns the port of the Mongodb server to use for testing.
// It reads the value from the MONGODB_PORT environment variable and returns
// 27017 if it is not set.
func GetEnvPort() string {
	port := os.Getenv("MONGODB_PORT")

	if len(port) == 0 {
		port = "27017"
	}
	return port
}
