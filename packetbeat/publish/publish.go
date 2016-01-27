package publish

import (
	"errors"
	"fmt"
	"sync"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"
)

type Transactions interface {
	PublishTransaction(common.MapStr) bool
}

type PacketbeatPublisher struct {
	pub    *publisher.PublisherType
	client publisher.Client

	wg     sync.WaitGroup
	events chan common.MapStr
	done   chan struct{}
}

type ChanTransactions struct {
	Channel chan common.MapStr
}

func (t *ChanTransactions) PublishTransaction(event common.MapStr) bool {
	t.Channel <- event
	return true
}

var debugf = logp.MakeDebug("publish")

func NewPublisher(pub *publisher.PublisherType, hwm int) *PacketbeatPublisher {
	return &PacketbeatPublisher{
		pub:    pub,
		client: pub.Client(),
		done:   make(chan struct{}),
		events: make(chan common.MapStr, hwm),
	}
}

func (t *PacketbeatPublisher) PublishTransaction(event common.MapStr) bool {
	select {
	case t.events <- event:
		return true
	default:
		return false
	}
}

func (t *PacketbeatPublisher) Start() {
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()

		for {
			select {
			case event := <-t.events:
				t.onEvent(event)
			case <-t.done:
				return
			}
		}
	}()
}

func (t *PacketbeatPublisher) Stop() {
	close(t.done)
	t.wg.Wait()
}

func (t *PacketbeatPublisher) onEvent(event common.MapStr) {
	if err := validateEvent(event); err != nil {
		logp.Warn("Dropping invalid event: %v", err)
		return
	}

	debugf("on event")

	if !updateEventAddresses(t.pub, event) {
		return
	}

	t.client.PublishEvent(event)
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

	err := event.EnsureCountField()
	if err != nil {
		return err
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

func updateEventAddresses(pub *publisher.PublisherType, event common.MapStr) bool {
	var srcServer, dstServer string
	src, ok := event["src"].(*common.Endpoint)

	if ok {
		// check if it's outgoing transaction (as client)
		isOutgoing := pub.IsPublisherIP(src.Ip)
		if isOutgoing {
			if pub.IgnoreOutgoing {
				// duplicated transaction -> ignore it
				debugf("Ignore duplicated transaction on: %s -> %s", srcServer, dstServer)
				return false
			}

			//outgoing transaction
			event["direction"] = "out"
		}

		srcServer = pub.GetServerName(src.Ip)
		event["client_ip"] = src.Ip
		event["client_port"] = src.Port
		event["client_proc"] = src.Proc
		event["client_server"] = srcServer
		delete(event, "src")
	}

	dst, ok := event["dst"].(*common.Endpoint)
	if ok {
		dstServer = pub.GetServerName(dst.Ip)
		event["ip"] = dst.Ip
		event["port"] = dst.Port
		event["proc"] = dst.Proc
		event["server"] = dstServer
		delete(event, "dst")

		//check if it's incoming transaction (as server)
		if pub.IsPublisherIP(dst.Ip) {
			// incoming transaction
			event["direction"] = "in"
		}

	}

	event.EnsureCountField()

	if pub.GeoLite != nil {
		realIP, exists := event["real_ip"]
		if exists && len(realIP.(common.NetString)) > 0 {
			loc := pub.GeoLite.GetLocationByIP(string(realIP.(common.NetString)))
			if loc != nil && loc.Latitude != 0 && loc.Longitude != 0 {
				loc := fmt.Sprintf("%f, %f", loc.Latitude, loc.Longitude)
				event["client_location"] = loc
			}
		} else {
			if len(srcServer) == 0 && src != nil { // only for external IP addresses
				loc := pub.GeoLite.GetLocationByIP(src.Ip)
				if loc != nil && loc.Latitude != 0 && loc.Longitude != 0 {
					loc := fmt.Sprintf("%f, %f", loc.Latitude, loc.Longitude)
					event["client_location"] = loc
				}
			}
		}
	}

	return true
}
