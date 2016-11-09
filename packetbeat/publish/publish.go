package publish

import (
	"errors"
	"fmt"
	"sync"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/nranchev/go-libGeoIP"
)

type Transactions interface {
	PublishTransaction(common.MapStr) bool
}

type Flows interface {
	PublishFlows([]common.MapStr) bool
}

type PacketbeatPublisher struct {
	pub    publisher.Publisher
	client publisher.Client

	topo           topologyProvider
	geoLite        *libgeo.GeoIP
	ignoreOutgoing bool

	wg   sync.WaitGroup
	done chan struct{}

	trans chan common.MapStr
	flows chan []common.MapStr
}

type ChanTransactions struct {
	Channel chan common.MapStr
}

// XXX: currently implemented by libbeat publisher. This functionality is only
// required by packetbeat. Source for TopologyProvider should become local to
// packetbeat.
type topologyProvider interface {
	IsPublisherIP(ip string) bool
	GetServerName(ip string) string
	GeoLite() *libgeo.GeoIP
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
	topo, ok := pub.(topologyProvider)
	if !ok {
		return nil, errors.New("Requires topology provider")
	}

	return &PacketbeatPublisher{
		pub:            pub,
		topo:           topo,
		geoLite:        topo.GeoLite(),
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

		if !p.addGeoIPToFlow(event) {
			continue
		}

		pub = append(pub, event)
	}

	p.client.PublishEvents(pub)
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
		isOutgoing := p.topo.IsPublisherIP(src.IP)
		if isOutgoing {
			if p.ignoreOutgoing {
				// duplicated transaction -> ignore it
				debugf("Ignore duplicated transaction on: %s -> %s", srcServer, dstServer)
				return false
			}

			//outgoing transaction
			event["direction"] = "out"
		}

		srcServer = p.topo.GetServerName(src.IP)
		event["client_ip"] = src.IP
		event["client_port"] = src.Port
		event["client_proc"] = src.Proc
		event["client_server"] = srcServer
		delete(event, "src")
	}

	dst, ok := event["dst"].(*common.Endpoint)
	debugf("has dst: %v", ok)
	if ok {
		dstServer = p.topo.GetServerName(dst.IP)
		event["ip"] = dst.IP
		event["port"] = dst.Port
		event["proc"] = dst.Proc
		event["server"] = dstServer
		delete(event, "dst")

		//check if it's incoming transaction (as server)
		if p.topo.IsPublisherIP(dst.IP) {
			// incoming transaction
			event["direction"] = "in"
		}

	}

	if p.geoLite != nil {
		realIP, exists := event["real_ip"]
		if exists && len(realIP.(common.NetString)) > 0 {
			loc := p.geoLite.GetLocationByIP(string(realIP.(common.NetString)))
			if loc != nil && loc.Latitude != 0 && loc.Longitude != 0 {
				loc := fmt.Sprintf("%f, %f", loc.Latitude, loc.Longitude)
				event["client_location"] = loc
			}
		} else {
			if len(srcServer) == 0 && src != nil { // only for external IP addresses
				loc := p.geoLite.GetLocationByIP(src.IP)
				if loc != nil && loc.Latitude != 0 && loc.Longitude != 0 {
					loc := fmt.Sprintf("%f, %f", loc.Latitude, loc.Longitude)
					event["client_location"] = loc
				}
			}
		}
	}

	return true
}

func (p *PacketbeatPublisher) addGeoIPToFlow(event common.MapStr) bool {

	getLocation := func(host common.MapStr, ip_type string) string {

		ip, exists := host[ip_type]
		if !exists {
			return ""
		}

		str, ok := ip.(string)
		if !ok {
			logp.Warn("IP address must be string")
			return ""
		}
		loc := p.geoLite.GetLocationByIP(str)
		if loc == nil || loc.Latitude == 0 || loc.Longitude == 0 {
			return ""
		}

		return fmt.Sprintf("%f, %f", loc.Latitude, loc.Longitude)
	}

	if p.geoLite == nil {
		return true
	}

	ipFieldNames := [][]string{
		{"ip", "ip_location"},
		{"outter_ip", "outter_ip_location"},
		{"ipv6", "ipv6_location"},
		{"outter_ipv6", "outter_ipv6_location"},
	}

	source := event["source"].(common.MapStr)
	dest := event["dest"].(common.MapStr)

	for _, name := range ipFieldNames {

		loc := getLocation(source, name[0])
		if loc != "" {
			source[name[1]] = loc
		}

		loc = getLocation(dest, name[0])
		if loc != "" {
			dest[name[1]] = loc
		}
	}
	event["source"] = source
	event["dest"] = dest

	return true
}
