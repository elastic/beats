package logstash

import "os"

// GetEnvHost for Logstash
func GetEnvHost() string {
	host := os.Getenv("LOGSTASH_HOST")

	if len(host) == 0 {
		host = "127.0.0.1"
	}
	return host
}

// GetEnvPort for Logstash
func GetEnvPort() string {
	port := os.Getenv("LOGSTASH_PORT")

	if len(port) == 0 {
		port = "9600"
	}
	return port
}

// GetConfig for Logstash
func GetConfig(metricset string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "logstash",
		"metricsets": []string{metricset},
		"hosts":      []string{GetEnvHost() + ":" + GetEnvPort()},
	}
}
