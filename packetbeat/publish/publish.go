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

type Flows interface {
	PublishFlows([]common.MapStr) bool
}

type PacketbeatPublisher struct {
	pub    *publisher.PublisherType
	client publisher.Client

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

func NewPublisher(pub *publisher.PublisherType, hwm, bulkHWM int) *PacketbeatPublisher {
	return &PacketbeatPublisher{
		pub:    pub,
		client: pub.Client(),
		done:   make(chan struct{}),
		trans:  make(chan common.MapStr, hwm),
		flows:  make(chan []common.MapStr, bulkHWM),
	}
}

func (t *PacketbeatPublisher) PublishTransaction(event common.MapStr) bool {
	select {
	case t.trans <- event:
		return true
	default:
		// drop event if queue is full
		return false
	}
}

func (t *PacketbeatPublisher) PublishFlows(event []common.MapStr) bool {
	select {
	case t.flows <- event:
		return true
	case <-t.done:
		// drop event, if worker has been stopped
		return false
	}
}

func (t *PacketbeatPublisher) Start() {
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		for {
			select {
			case <-t.done:
				return
			case event := <-t.trans:
				t.onTransaction(event)
			}
		}
	}()

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		for {
			select {
			case <-t.done:
				return
			case events := <-t.flows:
				t.onFlow(events)
			}
		}
	}()
}

func (t *PacketbeatPublisher) Stop() {
	close(t.done)
	t.wg.Wait()
}

func (t *PacketbeatPublisher) onTransaction(event common.MapStr) {
	if err := validateEvent(event); err != nil {
		logp.Warn("Dropping invalid event: %v", err)
		return
	}

	if !normalizeTransAddr(t.pub, event) {
		return
	}

	t.client.PublishEvent(event)
}

func (t *PacketbeatPublisher) onFlow(events []common.MapStr) {
	pub := events[:0]
	for _, event := range events {
		if err := validateEvent(event); err != nil {
			logp.Warn("Dropping invalid event: %v", err)
			continue
		}

		if !addGeoIPToFlow(t.pub, event) {
			continue
		}

		pub = append(pub, event)
	}

	t.client.PublishEvents(pub)
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

func normalizeTransAddr(pub *publisher.PublisherType, event common.MapStr) bool {
	debugf("normalize address for: %v", event)

	var srcServer, dstServer string
	src, ok := event["src"].(*common.Endpoint)
	debugf("has src: %v", ok)
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
	debugf("has dst: %v", ok)
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

func addGeoIPToFlow(pub *publisher.PublisherType, event common.MapStr) bool {
	if pub.GeoLite == nil {
		return true
	}

	ipFieldNames := [][]string{
		{"ip4_source", "ip4_source_location"},
		{"ip4_dest", "ip4_dest_location"},
		{"outter_ip4_source", "outter_ip4_source_location"},
		{"outter_ip4_dest", "outter_ip4_dest_location"},
		{"ip6_source", "ip6_source_location"},
		{"ip6_dest", "ip6_dest_location"},
		{"outter_ip6_source", "outter_ip6_source_location"},
		{"outter_ip6_dest", "outter_ip6_dest_location"},
	}

	for _, name := range ipFieldNames {
		ip, exists := event[name[0]]
		if !exists {
			continue
		}

		str, ok := ip.(string)
		if !ok {
			logp.Warn("IP address must be string")
			return false
		}

		loc := pub.GeoLite.GetLocationByIP(str)
		if loc == nil || loc.Latitude == 0 || loc.Longitude == 0 {
			continue
		}

		event[name[1]] = fmt.Sprintf("%f, %f", loc.Latitude, loc.Longitude)
	}

	return true
}
