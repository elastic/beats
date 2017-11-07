package elasticsearch

import "os"

func GetEnvHost() string {
	host := os.Getenv("ELASTICSEARCH_HOST")

	if len(host) == 0 {
		host = "127.0.0.1"
	}
	return host
}

func GetEnvPort() string {
	port := os.Getenv("ELASTICSEARCH_PORT")

	if len(port) == 0 {
		port = "9200"
	}
	return port
}

func GetConfig(metricset string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "elasticsearch",
		"metricsets": []string{metricset},
		"hosts":      []string{GetEnvHost() + ":" + GetEnvPort()},
	}
}
