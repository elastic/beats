package jolokia

import (
	"encoding/json"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/libbeat/logp"
)

// Jolokia Discovery query
// {
//   "type": "query"
// }
//
// Example Jolokia Discovery response
// {
//   "agent_version": "1.5.0",
//   "agent_id": "172.18.0.2-7-1322ae88-servlet",
//   "server_product": "tomcat",
//   "type": "response",
//   "server_vendor": "Apache",
//   "server_version": "7.0.86",
//   "secured": false,
//   "url": "http://172.18.0.2:8778/jolokia"
// }
//
// Example discovery probe with socat
//
//   echo '{"type":"query"}' | sudo socat STDIO UDP4-DATAGRAM:239.192.48.84:24884,interface=br0 | jq .
//

// Message contains the information of a Jolokia Discovery message
var messageSchema = s.Schema{
	"agent": s.Object{
		"id":      c.Str("agent_id"),
		"version": c.Str("agent_version", s.Optional),
	},
	"secured": c.Bool("secured", s.Optional),
	"server": s.Object{
		"product": c.Str("server_product", s.Optional),
		"vendor":  c.Str("server_vendor", s.Optional),
		"version": c.Str("server_version", s.Optional),
	},
	"url": c.Str("url"),
}

type Event struct {
	Type    string
	Message common.MapStr
}

func (e *Event) BusEvent() bus.Event {
	event := bus.Event{
		e.Type:    true,
		"host":    e.Message["url"],
		"jolokia": e.Message,
		"meta": common.MapStr{
			"jolokia": e.Message,
		},
	}
	return event
}

type Instance struct {
	LastSeen time.Time
	Message  common.MapStr
}

type Discovery struct {
	sync.Mutex

	Interfaces   []string
	Period       time.Duration
	ProbeTimeout time.Duration
	GracePeriod  time.Duration

	instances map[string]*Instance

	events chan Event
	stop   chan struct{}
}

func (d *Discovery) Start() {
	d.instances = make(map[string]*Instance)
	d.events = make(chan Event)
	d.stop = make(chan struct{})
	go d.run()
}

func (d *Discovery) Stop() {
	d.stop <- struct{}{}
	close(d.events)
}

func (d *Discovery) Events() chan Event {
	return d.events
}

func (d *Discovery) run() {
	for {
		d.sendProbe()
		d.checkStopped()

		select {
		case <-time.After(d.Period):
		case <-d.stop:
			return
		}
	}
}

// TODO: Check if this can be reused, or if packetbeat has something for this
func (d *Discovery) interfaces() ([]net.Interface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	matching := make(map[string]net.Interface)
	for _, name := range d.Interfaces {
		if name == "any" {
			return interfaces, nil
		}

		for _, i := range interfaces {
			if _, found := matching[i.Name]; !found && matchInterfaceName(name, i.Name) {
				matching[i.Name] = i
			}
		}
	}

	r := make([]net.Interface, 0, len(matching))
	for _, i := range matching {
		r = append(r, i)
	}
	return r, nil
}

func matchInterfaceName(name, candidate string) bool {
	if strings.HasSuffix(name, "*") {
		return strings.HasPrefix(candidate, strings.TrimRight(name, "*"))
	}
	return name == candidate
}

func getIPv4Addr(i net.Interface) (net.IP, error) {
	addrs, err := i.Addrs()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get addresses for "+i.Name)
	}
	for _, a := range addrs {
		if ip, _, err := net.ParseCIDR(a.String()); err == nil && ip != nil {
			if ipv4 := ip.To4(); ipv4 != nil {
				return ipv4, nil
			}
		}
	}
	return nil, nil
}

var discoveryAddress = net.UDPAddr{IP: net.IPv4(239, 192, 48, 84), Port: 24884}
var queryMessage = []byte(`{"type":"query"}`)

func (d *Discovery) sendProbe() {
	interfaces, err := d.interfaces()
	if err != nil {
		logp.Err("failed to get interfaces: ", err)
		return
	}

	var wg sync.WaitGroup
	for _, i := range interfaces {
		ip, err := getIPv4Addr(i)
		if err != nil {
			logp.Err(err.Error())
			continue
		}
		if ip == nil {
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

			conn, err := net.ListenPacket("udp4", net.JoinHostPort(ip.String(), "0"))
			if err != nil {
				logp.Err(err.Error())
				return
			}
			defer conn.Close()
			conn.SetDeadline(time.Now().Add(d.ProbeTimeout))

			if _, err := conn.WriteTo(queryMessage, &discoveryAddress); err != nil {
				logp.Err(err.Error())
				return
			}

			b := make([]byte, 1500)
			for {
				n, _, err := conn.ReadFrom(b)
				if err != nil {
					if !err.(net.Error).Timeout() {
						logp.Err(err.Error())
					}
					return
				}
				m := make(map[string]interface{})
				err = json.Unmarshal(b[:n], &m)
				if err != nil {
					logp.Err(err.Error())
					continue
				}
				message, _ := messageSchema.Apply(m)
				/*
					if err != nil {
						logp.Err(err.Error())
						continue
					}
				*/
				d.update(message)
			}
		}()
	}
	wg.Wait()
}

func (d *Discovery) update(message common.MapStr) {
	v, err := message.GetValue("agent.id")
	if err != nil {
		logp.Err("failed to update agent without id: " + err.Error())
		return
	}
	agentId, ok := v.(string)
	if len(agentId) == 0 || !ok {
		logp.Err("empty agent?")
		return
	}

	d.Lock()
	defer d.Unlock()
	i, found := d.instances[agentId]
	if !found {
		i = &Instance{Message: message}
		d.instances[agentId] = i
		d.events <- Event{"start", message}
	}
	i.LastSeen = time.Now()
}

func (d *Discovery) checkStopped() {
	d.Lock()
	defer d.Unlock()

	for id, i := range d.instances {
		if time.Since(i.LastSeen) > d.GracePeriod {
			d.events <- Event{"stop", i.Message}
			delete(d.instances, id)
		}
	}
}
