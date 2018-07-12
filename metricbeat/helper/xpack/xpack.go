package xpack

import (
	"fmt"
	"time"
)

// Product supported by X-Pack Monitoring
type Product int

const (
	// Elasticsearch product
	Elasticsearch Product = iota

	// Kibana product
	Kibana

	// Logstash product
	Logstash

	// Beats product
	Beats
)

func (p Product) String() string {
	indexProductNames := []string{
		"es",
		"kibana",
		"logstash",
		"beats",
	}

	if int(p) < 0 || int(p) > len(indexProductNames) {
		panic("Unknown product")
	}

	return indexProductNames[p]
}

// MakeMonitoringIndexName method returns the name of the monitoring index for
// a given product { elasticsearch, kibana, logstash, beats }
func MakeMonitoringIndexName(product Product) string {
	today := time.Now().UTC().Format("2006.01.02")
	const version = "6"

	return fmt.Sprintf(".monitoring-%v-%v-mb-%v", product, version, today)
}
