package redis

import (
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/metricbeat/helper"

	"github.com/garyburd/redigo/redis"

	"os"
)

func init() {
	Module.Register()
}

var Module = helper.NewModule("redis", Redis{})

var Config = &RedisModuleConfig{}

type RedisModuleConfig struct {
	Metrics map[string]interface{}
	Hosts   []string
}

type Redis struct {
	Name   string
	Config RedisModuleConfig
}

func (r Redis) Setup() error {
	// Loads module config
	// This is module specific config object
	Module.LoadConfig(&Config)
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

///*** Helper functions for testing ***///

func GetRedisEnvHost() string {
	host := os.Getenv("REDIS_HOST")

	if len(host) == 0 {
		host = "127.0.0.1"
	}
	return host
}

func GetRedisEnvPort() string {
	port := os.Getenv("REDIS_PORT")

	if len(port) == 0 {
		port = "6379"
	}
	return port
}
