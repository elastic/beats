package udpjson

import (
	"encoding/json"
	"net"
	"packetbeat/common"
	"packetbeat/config"
	"packetbeat/logp"
	"time"
)

type Config struct {
	Port    int
	BindIp  string
	Timeout time.Duration
}

type Udpjson struct {
	Config

	events  chan common.MapStr
	isAlive bool
	conn    *net.UDPConn
}

func (server *Udpjson) Run() error {

	buf := make([]byte, 65535)

	for server.isAlive {
		err := server.conn.SetDeadline(time.Now().Add(server.Config.Timeout))
		if err != nil {
			logp.Err("SetDeadline: %v", err)
			return err
		}
		n, _, err := server.conn.ReadFromUDP(buf)
		if err != nil {
			if err.(net.Error).Timeout() {
				continue
			}
			logp.Err("ReadFromUDP: %v", err)
			return err
		}

		logp.Debug("udpjson", "Read from socket: %s", string(buf[:n]))

		var obj common.MapStr
		err = json.Unmarshal(buf[:n], &obj)
		if err != nil {
			logp.Warn("json.Unmarshal failed: %v", err)
			continue
		}

		server.events <- obj
	}
	return nil
}

func (server *Udpjson) setFromConfig() error {
	var cfg Config

	if len(config.ConfigSingleton.Udpjson.Bind_ip) > 0 {
		cfg.BindIp = config.ConfigSingleton.Udpjson.Bind_ip
	} else {
		cfg.BindIp = "127.0.0.1"
	}
	if config.ConfigSingleton.Udpjson.Port > 0 {
		cfg.Port = config.ConfigSingleton.Udpjson.Port
	} else {
		cfg.Port = 9712
	}
	if config.ConfigSingleton.Udpjson.Timeout > 0 {
		cfg.Timeout = time.Duration(config.ConfigSingleton.Udpjson.Timeout) * time.Millisecond
	} else {
		cfg.Timeout = 10 * time.Millisecond
	}

	server.Config = cfg
	return nil
}

func (server *Udpjson) Init(test_mode bool, events chan common.MapStr) error {

	if !test_mode {
		err := server.setFromConfig()
		if err != nil {
			return err
		}
	}

	server.events = events

	addr := net.UDPAddr{
		Port: server.Config.Port,
		IP:   net.ParseIP(server.Config.BindIp),
	}

	var err error
	server.conn, err = net.ListenUDP("udp", &addr)
	if err != nil {
		return err
	}
	server.isAlive = true

	return nil
}

func (server *Udpjson) Stop() error {
	server.isAlive = false
	return nil
}

func (server *Udpjson) Close() error {
	return server.conn.Close()
}
