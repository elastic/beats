package influxdb

import (
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
)

type influxdbOut struct {
	beat beat.Info
}

var debugf = logp.MakeDebug("influxdb")

const (
	defaultWaitRetry    = 1 * time.Second
	defaultMaxWaitRetry = 60 * time.Second
)


type fieldRole int
const (
  fieldRoleTag = iota
  fieldRoleTime = iota
)

func init() {
	outputs.RegisterType("influxdb", makeInfluxdb)
}


func makeInfluxdb(
	beat beat.Info,
	stats *outputs.Stats,
	cfg *common.Config,
) (outputs.Group, error) {
  var err error
	config := defaultConfig
	if err = cfg.Unpack(&config); err != nil {
		return outputs.Fail(err)
	}


	_, err = outputs.LoadTLSConfig(config.TLS)
	if err != nil {
		return outputs.Fail(err)
	}


  client := newClient(stats, config.Addr, config.Username, config.Password, 
      config.Db, config.Measurement, config.TimePrecision, config.SendAsTags, config.SendAsTime)

	return outputs.Success(config.BulkMaxSize, config.MaxRetries, client)
}
