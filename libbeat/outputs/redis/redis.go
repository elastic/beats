package redis

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/garyburd/redigo/redis"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
)

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

type redisDataType uint16

type message struct {
	trans outputs.Signaler
	index string
	msg   string
}

func init() {
	outputs.RegisterOutputPlugin("redis", New)
}

func New(cfg *common.Config, topologyExpire int) (outputs.Outputer, error) {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	output := &redisOutput{}
	if err := output.Init(&config, topologyExpire); err != nil {
		return nil, err
	}
	return output, nil
}

func (out *redisOutput) Init(config *redisConfig, topology_expire int) error {

	out.Hostname = fmt.Sprintf("%s:%d", config.Host, config.Port)
	out.Password = config.Password
	out.Index = config.Index
	out.Db = config.Db
	out.DbTopology = config.DbTopology

	out.Timeout = config.Timeout

	out.ReconnectInterval = time.Duration(1) * time.Second
	if config.ReconnectInterval >= 0 {
		out.ReconnectInterval = time.Duration(config.ReconnectInterval) * time.Second
	}
	logp.Info("Reconnect Interval set to: %v", out.ReconnectInterval)

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

func (out *redisOutput) Close() error {
	return out.Conn.Close()
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
	opts outputs.Options,
	event common.MapStr,
) error {
	return out.BulkPublish(signal, opts, []common.MapStr{event})
}

func (out *redisOutput) BulkPublish(
	signal outputs.Signaler,
	opts outputs.Options,
	events []common.MapStr,
) error {
	if !opts.Guaranteed {
		err := out.doBulkPublish(events)
		outputs.Signal(signal, err)
		return err
	}

	for {
		err := out.doBulkPublish(events)
		if err == nil {
			outputs.SignalCompleted(signal)
			return nil
		}

		// TODO: add backoff
		time.Sleep(1)
	}
}

func (out *redisOutput) doBulkPublish(events []common.MapStr) error {
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
			return err
		}

		_, err = out.Conn.Do(command, out.Index, string(jsonEvent))
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
			out.onFail(err)
			return err
		}
	}
	if err := out.Conn.Flush(); err != nil {
		out.onFail(err)
		return err
	}
	_, err := out.Conn.Receive()
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
