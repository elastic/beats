package publisher

import (
	"errors"
	"fmt"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/outputs"
)

type preprocessor struct {
	handler messageHandler
	pub     *PublisherType
}

func newPreprocessor(p *PublisherType, h messageHandler) *preprocessor {
	return &preprocessor{
		handler: h,
		pub:     p,
	}
}

func (p *preprocessor) onStop() { p.handler.onStop() }

func (p *preprocessor) onMessage(m message) {
	publisher := p.pub
	single := false
	events := m.events
	if m.event != nil {
		single = true
		events = []common.MapStr{m.event}
	}

	var ignore []int // indices of events to be removed from events

	debug("Start Preprocessing")

	for i, event := range events {
		// validate some required field
		if err := filterEvent(event); err != nil {
			logp.Err("Publishing event failed: %v", err)
			ignore = append(ignore, i)
			continue
		}

		// update address and geo-ip information. Ignore event
		// if address is invalid or event is found to be a duplicate
		ok := updateEventAddresses(publisher, event)
		if !ok {
			ignore = append(ignore, i)
			continue
		}

		// add additional Beat meta data
		event["beat"] = common.MapStr{
			"name":     publisher.name,
			"hostname": publisher.hostname,
		}
		if len(publisher.tags) > 0 {
			event["tags"] = publisher.tags
		}

		if logp.IsDebug("publish") {
			PrintPublishEvent(event)
		}
	}

	// return if no event is left
	if len(ignore) == len(events) {
		debug("no event left, complete send")
		outputs.SignalCompleted(m.context.signal)
		return
	}

	// remove invalid events.
	// TODO: is order important? Removal can be turned into O(len(ignore)) by
	//       copying last element into idx and doing
	//       events=events[:len(events)-len(ignore)] afterwards
	// Alternatively filtering could be implemented like:
	//   https://github.com/golang/go/wiki/SliceTricks#filtering-without-allocating
	for i := len(ignore) - 1; i >= 0; i-- {
		idx := ignore[i]
		debug("remove event[%v]", idx)
		events = append(events[:idx], events[idx+1:]...)
	}

	if publisher.disabled {
		debug("publisher disabled")
		outputs.SignalCompleted(m.context.signal)
		return
	}

	debug("Forward preprocessed events")
	if single {
		p.handler.onMessage(message{context: m.context, event: events[0]})
	} else {
		p.handler.onMessage(message{context: m.context, events: events})
	}
}

// filterEvent validates an event for common required fields with types.
// If event is to be filtered out the reason is returned as error.
func filterEvent(event common.MapStr) error {
	ts, ok := event["@timestamp"]
	if !ok {
		return errors.New("Missing '@timestamp' field from event")
	}

	_, ok = ts.(common.Time)
	if !ok {
		return errors.New("Invalid '@timestamp' field from event.")
	}

	err := event.EnsureCountField()
	if err != nil {
		return err
	}

	t, ok := event["type"]
	if !ok {
		return errors.New("Missing 'type' field from event.")
	}

	_, ok = t.(string)
	if !ok {
		return errors.New("Invalid 'type' field from event.")
	}

	return nil
}

func updateEventAddresses(publisher *PublisherType, event common.MapStr) bool {
	var srcServer, dstServer string
	src, ok := event["src"].(*common.Endpoint)
	if ok {
		srcServer = publisher.GetServerName(src.Ip)
		event["client_ip"] = src.Ip
		event["client_port"] = src.Port
		event["client_proc"] = src.Proc
		event["client_server"] = srcServer
		delete(event, "src")
	}
	dst, ok := event["dst"].(*common.Endpoint)
	if ok {
		dstServer = publisher.GetServerName(dst.Ip)
		event["ip"] = dst.Ip
		event["port"] = dst.Port
		event["proc"] = dst.Proc
		event["server"] = dstServer
		delete(event, "dst")

		//get the direction of the transaction: outgoing (as client)/incoming (as server)
		if publisher.IsPublisherIP(dst.Ip) {
			// incoming transaction
			event["direction"] = "in"
		} else {
			//outgoing transaction
			event["direction"] = "out"
		}
	}

	if publisher.IgnoreOutgoing && dstServer != "" &&
		dstServer != publisher.name {
		// duplicated transaction -> ignore it
		debug("Ignore duplicated transaction on %s: %s -> %s",
			publisher.name, srcServer, dstServer)
		return false
	}

	if publisher.GeoLite != nil {
		realIP, exists := event["real_ip"]
		if exists && len(realIP.(string)) > 0 {
			loc := publisher.GeoLite.GetLocationByIP(realIP.(string))
			if loc != nil && loc.Latitude != 0 && loc.Longitude != 0 {
				loc := fmt.Sprintf("%f, %f", loc.Latitude, loc.Longitude)
				event["client_location"] = loc
			}
		} else {
			if len(srcServer) == 0 && src != nil { // only for external IP addresses
				loc := publisher.GeoLite.GetLocationByIP(src.Ip)
				if loc != nil && loc.Latitude != 0 && loc.Longitude != 0 {
					loc := fmt.Sprintf("%f, %f", loc.Latitude, loc.Longitude)
					event["client_location"] = loc
				}
			}
		}
	}

	return true
}
