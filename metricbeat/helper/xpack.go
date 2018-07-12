package helper

import (
	"fmt"
	"time"
)

// MakeMonitoringIndexName method returns the name of the monitoring index for
// a given product { elasticsearch, kibana, logstash, beats }
func MakeMonitoringIndexName(product string) string {
	if !(product == "elasticsearch" || product == "kibana" || product == "logstash" || product == "beats") {
		panic(fmt.Sprintf("Invalid product '%v'", product))
	}

	today := time.Now().UTC().Format("2006.01.02")
	const version = "6"

	return fmt.Sprintf(".monitoring-%v-%v-mb-%v", product, version, today)
}
