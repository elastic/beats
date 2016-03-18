package redis

import (
	"github.com/garyburd/redigo/redis"

	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/metricbeat/helper"
)

func init() {
	helper.Registry.AddModuler("redis", New)
}

// New creates new instance of Moduler
func New() helper.Moduler {
	return &Moduler{}
}

type Moduler struct{}

func (m *Moduler) Setup(mo *helper.Module) error {
	return nil
}

func Connect(host string) (redis.Conn, error) {

	conn, err := redis.Dial("tcp", host)
	if err != nil {
		logp.Err("Redis connection error: %v", err)
	}

	//defer conn.Close()
	return conn, err
}
