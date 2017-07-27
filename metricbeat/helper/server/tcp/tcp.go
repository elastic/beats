package tcp

import (
	"fmt"
	"net"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper/server"
	"github.com/elastic/beats/metricbeat/mb"
)

type TcpServer struct {
	listener          *net.TCPListener
	receiveBufferSize int
	done              chan struct{}
	eventQueue        chan server.Event
}

type TcpEvent struct {
	event common.MapStr
}

func (m *TcpEvent) GetEvent() common.MapStr {
	return m.event
}

func (m *TcpEvent) GetMeta() server.Meta {
	return server.Meta{}
}

func NewTcpServer(base mb.BaseMetricSet) (server.Server, error) {
	config := defaultTcpConfig()
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}

	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", config.Host, config.Port))

	if err != nil {
		return nil, err
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, err
	}

	logp.Info("Started listening for TCP on: %s:%d", config.Host, config.Port)
	return &TcpServer{
		listener:          listener,
		receiveBufferSize: config.ReceiveBufferSize,
		done:              make(chan struct{}),
		eventQueue:        make(chan server.Event),
	}, nil
}

func (g *TcpServer) Start() {
	go g.WatchMetrics()
}

func (g *TcpServer) WatchMetrics() {
	buffer := make([]byte, g.receiveBufferSize)
	for {
		select {
		case <-g.done:
			return
		default:
		}

		conn, err := g.listener.Accept()
		if err != nil {
			logp.Err("Unable to accept connection due to error: %v", err)
			continue
		}
		defer func() {
			if conn != nil {
				conn.Close()
			}
		}()

		length, err := conn.Read(buffer)
		if err != nil {
			logp.Err("Error reading from buffer: %v", err.Error())
			continue
		}
		g.eventQueue <- &TcpEvent{
			event: common.MapStr{
				server.EventDataKey: buffer[:length],
			},
		}
	}
}

func (g *TcpServer) GetEvents() chan server.Event {
	return g.eventQueue
}

func (g *TcpServer) Stop() {
	close(g.done)
	g.listener.Close()
	close(g.eventQueue)
}
