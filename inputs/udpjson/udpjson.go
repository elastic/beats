package udpjson

import (
	"encoding/json"
	"net"
	"packetbeat/common"
	"packetbeat/logp"
	"time"
)

type Config struct {
	Port   int
	BindIp string
}

type Server struct {
	Config

	events  chan common.MapStr
	isAlive bool
	timeout time.Duration
	conn    *net.UDPConn
}

func (server *Server) ReceiveForever() error {

	buf := make([]byte, 65535)

	for server.isAlive {
		err := server.conn.SetDeadline(time.Now().Add(server.timeout))
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

func (server *Server) Stop() {
	server.isAlive = false
}

func (server *Server) Close() {
	server.conn.Close()
}

func NewServer(config Config, timeout time.Duration, events chan common.MapStr) (*Server, error) {

	server := &Server{
		Config:  config,
		events:  events,
		isAlive: true,
		timeout: timeout,
	}

	addr := net.UDPAddr{
		Port: server.Config.Port,
		IP:   net.ParseIP(server.Config.BindIp),
	}

	var err error
	server.conn, err = net.ListenUDP("udp", &addr)
	if err != nil {
		return nil, err
	}

	return server, nil
}
