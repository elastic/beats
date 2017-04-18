package publish

import (
	"errors"
	"sync"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"
)

type Transactions interface {
	PublishTransaction(common.MapStr) bool
}

type Flows interface {
	PublishFlows([]common.MapStr) bool
}

type PacketbeatPublisher struct {
	beatPublisher *publisher.BeatPublisher
	client        publisher.Client

	ignoreOutgoing bool

	wg   sync.WaitGroup
	done chan struct{}

	trans chan common.MapStr
	flows chan []common.MapStr
}

type ChanTransactions struct {
	Channel chan common.MapStr
}

func (t *ChanTransactions) PublishTransaction(event common.MapStr) bool {
	t.Channel <- event
	return true
}

var debugf = logp.MakeDebug("publish")

func NewPublisher(
	pub publisher.Publisher,
	hwm, bulkHWM int,
	ignoreOutgoing bool,
) (*PacketbeatPublisher, error) {

	return &PacketbeatPublisher{
		beatPublisher:  pub.(*publisher.BeatPublisher),
		ignoreOutgoing: ignoreOutgoing,
		client:         pub.Connect(),
		done:           make(chan struct{}),
		trans:          make(chan common.MapStr, hwm),
		flows:          make(chan []common.MapStr, bulkHWM),
	}, nil
}

func (p *PacketbeatPublisher) PublishTransaction(event common.MapStr) bool {
	select {
	case p.trans <- event:
		return true
	default:
		// drop event if queue is full
		return false
	}
}

func (p *PacketbeatPublisher) PublishFlows(event []common.MapStr) bool {
	select {
	case p.flows <- event:
		return true
	case <-p.done:
		// drop event, if worker has been stopped
		return false
	}
}

func (p *PacketbeatPublisher) Start() {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		for {
			select {
			case <-p.done:
				return
			case event := <-p.trans:
				p.onTransaction(event)
			}
		}
	}()

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		for {
			select {
			case <-p.done:
				return
			case events := <-p.flows:
				p.onFlow(events)
			}
		}
	}()
}

func (p *PacketbeatPublisher) Stop() {
	p.client.Close()
	close(p.done)
	p.wg.Wait()
}

func (p *PacketbeatPublisher) onTransaction(event common.MapStr) {
	if err := validateEvent(event); err != nil {
		logp.Warn("Dropping invalid event: %v", err)
		return
	}

	if !p.normalizeTransAddr(event) {
		return
	}

	p.client.PublishEvent(event)
}

func (p *PacketbeatPublisher) onFlow(events []common.MapStr) {
	pub := events[:0]
	for _, event := range events {
		if err := validateEvent(event); err != nil {
			logp.Warn("Dropping invalid event: %v", err)
			continue
		}

		pub = append(pub, event)
	}

	p.client.PublishEvents(pub)
}

func (p *PacketbeatPublisher) IsPublisherIP(ip string) bool {

	for _, myip := range p.beatPublisher.IPAddrs {
		if myip == ip {
			return true
		}
	}

	return false
}

func (p *PacketbeatPublisher) GetServerName(ip string) string {

	// in case the IP is localhost, return current shipper name
	islocal, err := common.IsLoopback(ip)
	if err != nil {
		logp.Err("Parsing IP %s fails with: %s", ip, err)
		return ""
	}

	if islocal {
		return p.beatPublisher.GetName()
	}

	return ""
}

// filterEvent validates an event for common required fields with types.
// If event is to be filtered out the reason is returned as error.
func validateEvent(event common.MapStr) error {
	ts, ok := event["@timestamp"]
	if !ok {
		return errors.New("missing '@timestamp' field from event")
	}

	_, ok = ts.(common.Time)
	if !ok {
		return errors.New("invalid '@timestamp' field from event")
	}

	t, ok := event["type"]
	if !ok {
		return errors.New("missing 'type' field from event")
	}

	_, ok = t.(string)
	if !ok {
		return errors.New("invalid 'type' field from event")
	}

	return nil
}

func (p *PacketbeatPublisher) normalizeTransAddr(event common.MapStr) bool {
	debugf("normalize address for: %v", event)

	var srcServer, dstServer string
	src, ok := event["src"].(*common.Endpoint)
	debugf("has src: %v", ok)
	if ok {
		// check if it's outgoing transaction (as client)
		isOutgoing := p.IsPublisherIP(src.IP)
		if isOutgoing {
			if p.ignoreOutgoing {
				// duplicated transaction -> ignore it
				debugf("Ignore duplicated transaction on: %s -> %s", srcServer, dstServer)
				return false
			}

			//outgoing transaction
			event["direction"] = "out"
		}

		srcServer = p.GetServerName(src.IP)
		event["client_ip"] = src.IP
		event["client_port"] = src.Port
		event["client_proc"] = src.Proc
		event["client_server"] = srcServer
		delete(event, "src")
	}

	dst, ok := event["dst"].(*common.Endpoint)
	debugf("has dst: %v", ok)
	if ok {
		dstServer = p.GetServerName(dst.IP)
		event["ip"] = dst.IP
		event["port"] = dst.Port
		event["proc"] = dst.Proc
		event["server"] = dstServer
		delete(event, "dst")

		//check if it's incoming transaction (as server)
		if p.IsPublisherIP(dst.IP) {
			// incoming transaction
			event["direction"] = "in"
		}

	}

	return true
}
