package influxdb

import (
	"github.com/elastic/beats/libbeat/outputs"
)

type influxdbConfig struct {
	Username    string                `config:"username"`
	Password    string                `config:"password"`
	Addr        string                `config:"addr"`
	BulkMaxSize int                   `config:"bulk_max_size"`
	MaxRetries  int                   `config:"max_retries"`
	TLS         *outputs.TLSConfig    `config:"ssl"`
	Db          string                `config:"db"`
	Measurement string                `config:"measurement"`
 	TimePrecision string              `config:"time_precision"`
 	SendAsTags  []string              `config:"send_as_tags"` 
 	SendAsTime  string                `config:"send_as_time"` 
}

var (
	defaultConfig = influxdbConfig{
		Addr:        "http://localhost:8086",
		MaxRetries:  3,
		TLS:         nil,
		Db:          "test_db",
 		Measurement: "test",
 		TimePrecision: "s",
 		BulkMaxSize: 2048,
	}
)

func (c *influxdbConfig) Validate() error {
	return nil
}
