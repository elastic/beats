package mtest

import (
	"net"
	"os"
)

// GetEnvHost returns host for Kibana
func GetEnvHost() string {
	host := os.Getenv("KIBANA_HOST")

	if len(host) == 0 {
		host = "127.0.0.1"
	}
	return host
}

// GetEnvPort returns port for Kibana
func GetEnvPort() string {
	port := os.Getenv("KIBANA_PORT")

	if len(port) == 0 {
		port = "5601"
	}
	return port
}

// GetConfig returns config for kibana module
func GetConfig(metricset string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "kibana",
		"metricsets": []string{metricset},
		"hosts":      []string{net.JoinHostPort(GetEnvHost(), GetEnvPort())},
	}
}
