package udp

import (
	"fmt"
	"net"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper/server"
	"github.com/elastic/beats/metricbeat/mb"
)

type UdpServer struct {
	listener          *net.UDPConn
	receiveBufferSize int
	done              chan struct{}
	eventQueue        chan server.Event
}

type UdpEvent struct {
	event common.MapStr
	meta  server.Meta
}

func (u *UdpEvent) GetEvent() common.MapStr {
	return u.event
}

func (u *UdpEvent) GetMeta() server.Meta {
	return u.meta
}

func NewUdpServer(base mb.BaseMetricSet) (server.Server, error) {
	config := defaultUdpConfig()
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", config.Host, config.Port))

	if err != nil {
		return nil, err
	}

	listener, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	logp.Info("Started listening for UDP on: %s:%d", config.Host, config.Port)
	return &UdpServer{
		listener:          listener,
		receiveBufferSize: config.ReceiveBufferSize,
		done:              make(chan struct{}),
		eventQueue:        make(chan server.Event),
	}, nil
}

func (g *UdpServer) Start() {
	go g.WatchMetrics()
}

func (g *UdpServer) WatchMetrics() {
	buffer := make([]byte, g.receiveBufferSize)
	for {
		select {
		case <-g.done:
			return
		default:
		}

		length, addr, err := g.listener.ReadFromUDP(buffer)
		if err != nil {
			logp.Err("Error reading from buffer: %v", err.Error())
			continue
		}

		g.eventQueue <- &UdpEvent{
			event: common.MapStr{
				server.EventDataKey: buffer[:length],
			},
			meta: server.Meta{
				"client_ip": addr.IP.String(),
			},
		}
	}
}

func (g *UdpServer) GetEvents() chan server.Event {
	return g.eventQueue
}

func (g *UdpServer) Stop() {
	close(g.done)
	g.listener.Close()
	close(g.eventQueue)
}
