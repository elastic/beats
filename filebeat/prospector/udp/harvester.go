package udp

import (
	"net"
	"time"

	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/util"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher/beat"
)

type Harvester struct {
	forwarder *harvester.Forwarder
	done      chan struct{}
	cfg       *common.Config
	listener  net.PacketConn
}

func NewHarvester(forwarder *harvester.Forwarder, cfg *common.Config) *Harvester {
	return &Harvester{
		done:      make(chan struct{}),
		cfg:       cfg,
		forwarder: forwarder,
	}
}

func (h *Harvester) Run() error {

	config := defaultConfig
	err := h.cfg.Unpack(&config)
	if err != nil {
		return err
	}

	h.listener, err = net.ListenPacket("udp", config.Host)
	if err != nil {
		return err
	}
	defer h.listener.Close()

	logp.Info("Started listening for udp on: %s", config.Host)

	buffer := make([]byte, config.MaxMessageSize)

	for {
		select {
		case <-h.done:
			return nil
		default:
		}

		length, _, err := h.listener.ReadFrom(buffer)
		if err != nil {
			logp.Err("Error reading from buffer: %v", err.Error())
			continue
		}
		data := util.NewData()
		data.Event = beat.Event{
			Timestamp: time.Now(),
			Fields: common.MapStr{
				"message": string(buffer[:length]),
			},
		}
		h.forwarder.Send(data)
	}
}

func (h *Harvester) Stop() {
	logp.Info("Stopping udp harvester")
	close(h.done)
	h.listener.Close()
}
