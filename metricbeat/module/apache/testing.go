/*

Helper functions for testing used in the apache metricsets

*/
package apache

import (
	"os"
)

func GetApacheEnvHost() string {
	host := os.Getenv("APACHE_HOST")

	if len(host) == 0 {
		host = "127.0.0.1"
	}
	return host
}
