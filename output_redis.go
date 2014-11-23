package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
)

type RedisDataType uint16

const (
	RedisListType RedisDataType = iota
	RedisChannelType
)

type RedisOutputType struct {
	OutputInterface
	Index string
	Conn  redis.Conn

	TopologyExpire     time.Duration
	ReconnectInterval  time.Duration
	Hostname           string
	Password           string
	Db                 int
	DbTopology         int
	Timeout            time.Duration
	DataType           RedisDataType
	FlushInterval      time.Duration
	flush_immediatelly bool

	TopologyMap  map[string]string
	sendingQueue chan RedisQueueMsg
	connected    bool
}

type RedisQueueMsg struct {
	index string
	msg   string
}

var RedisOutput RedisOutputType

func (out *RedisOutputType) Init(config tomlMothership) error {

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
		out.Index = "packetbeat"
	}

	out.FlushInterval = 1000 * time.Millisecond
	if config.Flush_interval != 0 {
		if config.Flush_interval < 0 {
			out.flush_immediatelly = true
			WARN("Flushing to REDIS on each push, performance migh be affected")
		} else {
			out.FlushInterval = time.Duration(config.Flush_interval) * time.Millisecond
		}
	}

	out.ReconnectInterval = time.Duration(1) * time.Second
	if config.Reconnect_interval != 0 {
		out.ReconnectInterval = time.Duration(config.Reconnect_interval) * time.Second
	}

	exp_sec := 15
	if _Config.Agent.Topology_expire != 0 {
		exp_sec = _Config.Agent.Topology_expire
	}
	out.TopologyExpire = time.Duration(exp_sec) * time.Second

	switch config.DataType {
	case "", "list":
		out.DataType = RedisListType
	case "channel":
		out.DataType = RedisChannelType
	default:
		return errors.New("Bad Redis data type")
	}

	INFO("[RedisOutput] Using Redis server %s", out.Hostname)
	if out.Password != "" {
		INFO("[RedisOutput] Using password to connect to Redis")
	}
	INFO("[RedisOutput] Redis connection timeout %s", out.Timeout)
	INFO("[RedisOutput] Redis reconnect interval %s", out.ReconnectInterval)
	INFO("[RedisOutput] Redis flushing interval %s", out.FlushInterval)
	INFO("[RedisOutput] Using index pattern %s", out.Index)
	INFO("[RedisOutput] Topology expires after %s", out.TopologyExpire)
	INFO("[RedisOutput] Using db %d for storing events", out.Db)
	INFO("[RedisOutput] Using db %d for storing topology", out.DbTopology)
	INFO("[RedisOutput] Using %d data type", out.DataType)

	out.sendingQueue = make(chan RedisQueueMsg, 1000)

	out.Reconnect()
	go out.SendMessagesGoroutine()

	return nil
}

func (out *RedisOutputType) RedisConnect(db int) (redis.Conn, error) {
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

func (out *RedisOutputType) Connect() error {
	var err error
	out.Conn, err = out.RedisConnect(out.Db)
	if err != nil {
		return err
	}
	out.connected = true

	return nil
}

func (out *RedisOutputType) Close() {
	out.Conn.Close()
}

func (out *RedisOutputType) SendMessagesGoroutine() {

	var err error
	flushChannel := make(<-chan time.Time)

	if !out.flush_immediatelly {
		flushTimer := time.NewTimer(out.FlushInterval)
		flushChannel = flushTimer.C
	}

	for {
		select {
		case queueMsg := <-out.sendingQueue:

			if !out.connected {
				DEBUG("output_redis", "Droping pkt ...")
				continue
			}
			DEBUG("output_redis", "Send event to redis")
			command := "RPUSH"
			if out.DataType == RedisChannelType {
				command = "PUBLISH"
			}

			if !out.flush_immediatelly {
				err = out.Conn.Send(command, queueMsg.index, queueMsg.msg)
			} else {
				_, err = out.Conn.Do(command, queueMsg.index, queueMsg.msg)
			}
			if err != nil {
				ERR("Fail to publish event to REDIS: %s", err)
				out.connected = false
				go out.Reconnect()
			}
		case _ = <-flushChannel:
			out.Conn.Flush()
			_, err = out.Conn.Receive()
			if err != nil {
				ERR("Fail to publish event to REDIS: %s", err)
				out.connected = false
				go out.Reconnect()
			}
		}
	}
}

func (out *RedisOutputType) Reconnect() {

	for {
		err := out.Connect()
		if err != nil {
			WARN("Error connecting to Redis (%s). Retrying in %s", err, out.ReconnectInterval)
			time.Sleep(out.ReconnectInterval)
		} else {
			break
		}
	}
}

func (out *RedisOutputType) GetNameByIP(ip string) string {
	name, exists := out.TopologyMap[ip]
	if !exists {
		return ""
	}
	return name
}

func (out *RedisOutputType) PublishIPs(name string, localAddrs []string) error {

	DEBUG("output_redis", "[%s] Publish the IPs %s", name, localAddrs)

	// connect to db
	conn, err := out.RedisConnect(out.DbTopology)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Do("HSET", name, "ipaddrs", strings.Join(localAddrs, ","))
	if err != nil {
		ERR("[%s] Fail to set the IP addresses: %s", name, err)
		return err
	}

	_, err = conn.Do("EXPIRE", name, int(out.TopologyExpire.Seconds()))
	if err != nil {
		ERR("[%s] Fail to set the expiration time: %s", name, err)
		return err
	}

	out.UpdateLocalTopologyMap(conn)

	return nil
}

func (out *RedisOutputType) UpdateLocalTopologyMap(conn redis.Conn) {

	TopologyMapTmp := make(map[string]string)

	hostnames, err := redis.Strings(conn.Do("KEYS", "*"))
	if err != nil {
		ERR("Fail to get the all agents from the topology map %s", err)
		return
	}
	for _, hostname := range hostnames {
		res, err := redis.String(conn.Do("HGET", hostname, "ipaddrs"))
		if err != nil {
			ERR("[%s] Fail to get the IPs: %s", hostname, err)
		} else {
			ipaddrs := strings.Split(res, ",")
			for _, addr := range ipaddrs {
				TopologyMapTmp[addr] = hostname
			}
		}
	}

	out.TopologyMap = TopologyMapTmp

	DEBUG("output_redis", "Topology %s", TopologyMapTmp)
}

func (out *RedisOutputType) PublishEvent(event *Event) error {

	json_event, err := json.Marshal(event)
	if err != nil {
		ERR("Fail to convert the event to JSON: %s", err)
		return err
	}

	out.sendingQueue <- RedisQueueMsg{index: out.Index, msg: string(json_event)}

	DEBUG("output_redis", "Publish event")
	return nil
}
