package elasticsearch

import "os"

// GetEnvHost returns host for Elasticsearch
func GetEnvHost() string {
	host := os.Getenv("ES_HOST")

	if len(host) == 0 {
		host = "127.0.0.1"
	}
	return host
}

// GetEnvPort returns port for Elasticsearch
func GetEnvPort() string {
	port := os.Getenv("ES_PORT")

	if len(port) == 0 {
		port = "9200"
	}
	return port
}

// GetConfig returns config for elasticsearch module
func GetConfig(metricset string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "elasticsearch",
		"metricsets": []string{metricset},
		"hosts":      []string{GetEnvHost() + ":" + GetEnvPort()},
	}
}
