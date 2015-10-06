package redis

import (
	"os"
	"testing"
	"time"
)

const RedisDefaultHost = "localhost"
const RedisDefaultPort = "6379"

func GetRedisAddr() string {

	port := os.Getenv("REDIS_PORT")
	host := os.Getenv("REDIS_HOST")

	if len(port) == 0 {
		port = RedisDefaultPort
	}

	if len(host) == 0 {
		host = RedisDefaultHost
	}

	return host + ":" + port
}

func TestTopologyInRedis(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping topology tests in short mode, because they require REDIS")
	}

	var redisOutput1 = redisOutput{
		Index:          "packetbeat",
		Hostname:       GetRedisAddr(),
		Password:       "",
		DbTopology:     1,
		Timeout:        time.Duration(5) * time.Second,
		TopologyExpire: time.Duration(15) * time.Second,
	}

	var redisOutput2 = redisOutput{
		Index:          "packetbeat",
		Hostname:       GetRedisAddr(),
		Password:       "",
		DbTopology:     1,
		Timeout:        time.Duration(5) * time.Second,
		TopologyExpire: time.Duration(15) * time.Second,
	}

	var redisOutput3 = redisOutput{
		Index:          "packetbeat",
		Hostname:       GetRedisAddr(),
		Password:       "",
		DbTopology:     1,
		Timeout:        time.Duration(5) * time.Second,
		TopologyExpire: time.Duration(15) * time.Second,
	}

	redisOutput1.PublishIPs("proxy1", []string{"10.1.0.4"})
	redisOutput2.PublishIPs("proxy2", []string{"10.1.0.9", "fe80::4e8d:79ff:fef2:de6a"})
	redisOutput3.PublishIPs("proxy3", []string{"10.1.0.10"})

	name2 := redisOutput3.GetNameByIP("10.1.0.9")
	if name2 != "proxy2" {
		t.Errorf("Failed to update proxy2 in topology: name=%s", name2)
	}

	redisOutput1.PublishIPs("proxy1", []string{"10.1.0.4"})
	redisOutput2.PublishIPs("proxy2", []string{"10.1.0.9"})
	redisOutput3.PublishIPs("proxy3", []string{"192.168.1.2"})

	name3 := redisOutput3.GetNameByIP("192.168.1.2")
	if name3 != "proxy3" {
		t.Errorf("Failed to add a new IP")
	}

	name3 = redisOutput3.GetNameByIP("10.1.0.10")
	if name3 != "" {
		t.Errorf("Failed to delete old IP of proxy3: %s", name3)
	}

	name2 = redisOutput3.GetNameByIP("fe80::4e8d:79ff:fef2:de6a")
	if name2 != "" {
		t.Errorf("Failed to delete old IP of proxy2: %s", name2)
	}
}
