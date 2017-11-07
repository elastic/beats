package nginx

/*
Helper functions for testing used in the nginx metricsets
*/

import (
	"os"
)

func GetNginxEnvHost() string {
	host := os.Getenv("NGINX_HOST")

	if len(host) == 0 {
		host = "127.0.0.1"
	}
	return host
}
