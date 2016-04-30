/*
Package redis is a Metricbeat module for Redis servers.
*/
package redis

import (
	"github.com/elastic/beats/libbeat/logp"

	"github.com/garyburd/redigo/redis"
)

// Connect connects to the Redis server at the given host address.
func Connect(host string) (redis.Conn, error) {
	conn, err := redis.Dial("tcp", host)
	if err != nil {
		logp.Err("Redis connection error: %v", err)
	}

	return conn, err
}
