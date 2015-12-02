//@deprecated: Starting with version 1.0.0-beta4 the Redis Output is deprecated as
// it's replaced by the Logstash Output that has support for Redis Output plugin.

package redis

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/outputs"

	"github.com/garyburd/redigo/redis"
)

func init() {

	outputs.RegisterOutputPlugin("redis", RedisOutputPlugin{})
}

type RedisOutputPlugin struct{}

func (f RedisOutputPlugin) NewOutput(
	beat string,
	config *outputs.MothershipConfig,
	topology_expire int,
) (outputs.Outputer, error) {
	output := &redisOutput{}
	err := output.Init(beat, *config, topology_expire)
	if err != nil {
		return nil, err
	}
	return output, nil
}

type redisDataType uint16

const (
	RedisListType redisDataType = iota
	RedisChannelType
)

type redisOutput struct {
	Index string
	Conn  redis.Conn

	TopologyExpire    time.Duration
	ReconnectInterval time.Duration
	Hostname          string
	Password          string
	Db                int
	DbTopology        int
	Timeout           time.Duration
	DataType          redisDataType

	TopologyMap atomic.Value // Value holds a map[string][string]
	connected   bool
}

type message struct {
	trans outputs.Signaler
	index string
	msg   string
}

func (out *redisOutput) Init(beat string, config outputs.MothershipConfig, topology_expire int) error {

	logp.Warn("Redis Output is deprecated. Please use the Redis Output Plugin from Logstash instead.")

	out.Hostname = fmt.Sprintf("%s:%d", config.Host, config.Port)

	if config.Password != "" {
		out.Password = config.Password
	}

	if config.Db != 0 {
		out.Db = config.Db
	}

	out.DbTopology = 1
	if config.Db_topology != 0 {
		out.DbTopology = config.Db_topology
	}

	out.Timeout = 5 * time.Second
	if config.Timeout != 0 {
		out.Timeout = time.Duration(config.Timeout) * time.Second
	}

	if config.Index != "" {
		out.Index = config.Index
	} else {
		out.Index = beat
	}

	out.ReconnectInterval = time.Duration(1) * time.Second
	if config.Reconnect_interval != 0 {
		out.ReconnectInterval = time.Duration(config.Reconnect_interval) * time.Second
	}

	expSec := 15
	if topology_expire != 0 {
		expSec = topology_expire
	}
	out.TopologyExpire = time.Duration(expSec) * time.Second

	switch config.DataType {
	case "", "list":
		out.DataType = RedisListType
	case "channel":
		out.DataType = RedisChannelType
	default:
		return errors.New("Bad Redis data type")
	}

	logp.Info("[RedisOutput] Using Redis server %s", out.Hostname)
	if out.Password != "" {
		logp.Info("[RedisOutput] Using password to connect to Redis")
	}
	logp.Info("[RedisOutput] Redis connection timeout %s", out.Timeout)
	logp.Info("[RedisOutput] Redis reconnect interval %s", out.ReconnectInterval)
	logp.Info("[RedisOutput] Using index pattern %s", out.Index)
	logp.Info("[RedisOutput] Topology expires after %s", out.TopologyExpire)
	logp.Info("[RedisOutput] Using db %d for storing events", out.Db)
	logp.Info("[RedisOutput] Using db %d for storing topology", out.DbTopology)
	logp.Info("[RedisOutput] Using %d data type", out.DataType)

	out.Reconnect()

	return nil
}

func (out *redisOutput) RedisConnect(db int) (redis.Conn, error) {
	conn, err := redis.DialTimeout(
		"tcp",
		out.Hostname,
		out.Timeout, out.Timeout, out.Timeout)
	if err != nil {
		return nil, err
	}

	if len(out.Password) > 0 {
		_, err = conn.Do("AUTH", out.Password)
		if err != nil {
			return nil, err
		}
	}

	_, err = conn.Do("PING")
	if err != nil {
		return nil, err
	}

	_, err = conn.Do("SELECT", db)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (out *redisOutput) Connect() error {
	var err error
	out.Conn, err = out.RedisConnect(out.Db)
	if err != nil {
		return err
	}
	out.connected = true

	return nil
}

func (out *redisOutput) Close() {
	_ = out.Conn.Close()
}

func (out *redisOutput) Reconnect() {

	for {
		err := out.Connect()
		if err != nil {
			logp.Warn("Error connecting to Redis (%s). Retrying in %s", err, out.ReconnectInterval)
			time.Sleep(out.ReconnectInterval)
		} else {
			break
		}
	}
}

func (out *redisOutput) GetNameByIP(ip string) string {
	topologyMap, ok := out.TopologyMap.Load().(map[string]string)
	if ok {
		name, exists := topologyMap[ip]
		if exists {
			return name
		}
	}
	return ""
}

func (out *redisOutput) PublishIPs(name string, localAddrs []string) error {
	logp.Debug("output_redis", "[%s] Publish the IPs %s", name, localAddrs)

	// connect to db
	conn, err := out.RedisConnect(out.DbTopology)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	_, err = conn.Do("HSET", name, "ipaddrs", strings.Join(localAddrs, ","))
	if err != nil {
		logp.Err("[%s] Fail to set the IP addresses: %s", name, err)
		return err
	}

	_, err = conn.Do("EXPIRE", name, int(out.TopologyExpire.Seconds()))
	if err != nil {
		logp.Err("[%s] Fail to set the expiration time: %s", name, err)
		return err
	}

	out.UpdateLocalTopologyMap(conn)

	return nil
}

func (out *redisOutput) UpdateLocalTopologyMap(conn redis.Conn) {
	topologyMapTmp := make(map[string]string)
	hostnames, err := redis.Strings(conn.Do("KEYS", "*"))
	if err != nil {
		logp.Err("Fail to get the all shippers from the topology map %s", err)
		return
	}
	for _, hostname := range hostnames {
		res, err := redis.String(conn.Do("HGET", hostname, "ipaddrs"))
		if err != nil {
			logp.Err("[%s] Fail to get the IPs: %s", hostname, err)
		} else {
			ipaddrs := strings.Split(res, ",")
			for _, addr := range ipaddrs {
				topologyMapTmp[addr] = hostname
			}
		}
	}

	out.TopologyMap.Store(topologyMapTmp)

	logp.Debug("output_redis", "Topology %s", topologyMapTmp)
}

func (out *redisOutput) PublishEvent(
	signal outputs.Signaler,
	ts time.Time,
	event common.MapStr,
) error {
	return out.BulkPublish(signal, ts, []common.MapStr{event})
}

func (out *redisOutput) BulkPublish(
	signal outputs.Signaler,
	ts time.Time,
	events []common.MapStr,
) error {
	if !out.connected {
		logp.Debug("output_redis", "Droping pkt ...")
		return errors.New("Not connected")
	}

	command := "RPUSH"
	if out.DataType == RedisChannelType {
		command = "PUBLISH"
	}

	if len(events) == 1 { // single event
		event := events[0]
		jsonEvent, err := json.Marshal(event)
		if err != nil {
			logp.Err("Fail to convert the event to JSON: %s", err)
			outputs.SignalCompleted(signal)
			return err
		}

		_, err = out.Conn.Do(command, out.Index, string(jsonEvent))
		outputs.Signal(signal, err)
		out.onFail(err)
		return err
	}

	for _, event := range events {
		jsonEvent, err := json.Marshal(event)
		if err != nil {
			logp.Err("Fail to convert the event to JSON: %s", err)
			continue
		}
		err = out.Conn.Send(command, out.Index, string(jsonEvent))
		if err != nil {
			outputs.SignalFailed(signal, err)
			out.onFail(err)
			return err
		}
	}
	if err := out.Conn.Flush(); err != nil {
		outputs.Signal(signal, err)
		out.onFail(err)
		return err
	}
	_, err := out.Conn.Receive()
	outputs.Signal(signal, err)
	out.onFail(err)
	return err
}

func (out *redisOutput) onFail(err error) {
	if err != nil {
		logp.Err("Fail to publish event to REDIS: %s", err)
		out.connected = false
		go out.Reconnect()
	}
}
