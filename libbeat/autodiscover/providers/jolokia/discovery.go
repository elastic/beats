// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package jolokia

import (
	"encoding/json"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common/bus"
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
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
		"id":      c.Str("agent_id", s.Required),
		"version": c.Str("agent_version"),
	},
	"secured": c.Bool("secured"),
	"server": s.Object{
		"product": c.Str("server_product"),
		"vendor":  c.Str("server_vendor"),
		"version": c.Str("server_version"),
	},
	"url": c.Str("url"),
}

// Event is a Jolokia Discovery event
type Event struct {
	ProviderUUID uuid.UUID
	Type         string
	AgentID      string
	Message      mapstr.M
}

// BusEvent converts a Jolokia Discovery event to a autodiscover bus event
func (e *Event) BusEvent() bus.Event {
	event := bus.Event{
		e.Type:     true,
		"provider": e.ProviderUUID,
		"id":       e.AgentID,
		"jolokia":  e.Message,
		"meta": mapstr.M{
			"jolokia": e.Message,
		},
	}
	return event
}

// Instance is a discovered Jolokia instance, it keeps information of the
// last probe it replied
type Instance struct {
	LastSeen      time.Time
	LastInterface *InterfaceConfig
	AgentID       string
	Message       mapstr.M
}

// Discovery controls the Jolokia Discovery probes
type Discovery struct {
	sync.Mutex

	log *logp.Logger

	ProviderUUID uuid.UUID

	Interfaces []InterfaceConfig

	instances map[string]*Instance

	events chan Event
	stop   chan struct{}
}

// Start starts discovery probes
func (d *Discovery) Start() {
	d.instances = make(map[string]*Instance)
	d.events = make(chan Event)
	d.stop = make(chan struct{})
	if d.log == nil {
		d.log = logp.NewLogger("jolokia")
	}
	go d.run()
}

// Stop stops discovery probes
func (d *Discovery) Stop() {
	close(d.stop)
	close(d.events)
}

// Events returns a channel with the events of started and stopped Jolokia
// instances discovered
func (d *Discovery) Events() <-chan Event {
	return d.events
}

func (d *Discovery) run() {
	for _, i := range d.Interfaces {
		i := i
		go func() {
			for {
				d.sendProbe(i)
				d.checkStopped()
				select {
				case <-time.After(i.Interval):
				case <-d.stop:
					return
				}
			}
		}()
	}
	<-d.stop
}

func (d *Discovery) interfaces(name string) ([]net.Interface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	if name == "any" || name == "*" {
		return interfaces, nil
	}

	var matching []net.Interface
	for _, i := range interfaces {
		if matchInterfaceName(name, i.Name) {
			matching = append(matching, i)
		}
	}
	return matching, nil
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

func (d *Discovery) sendProbe(config InterfaceConfig) {
	log := d.log

	interfaces, err := d.interfaces(config.Name)
	if err != nil {
		log.Errorf("failed to get interfaces: %+v", err)
		return
	}

	var wg sync.WaitGroup
	for _, i := range interfaces {
		ip, err := getIPv4Addr(i)
		if err != nil {
			log.Error(err.Error())
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
				log.Error(err.Error())
				return
			}
			defer conn.Close()

			// Avoid having sockets open more time than needed
			timeout := config.ProbeTimeout
			if timeout > config.Interval {
				timeout = config.Interval
			}
			conn.SetDeadline(time.Now().Add(timeout))

			if _, err := conn.WriteTo(queryMessage, &discoveryAddress); err != nil {
				log.Error(err.Error())
				return
			}

			b := make([]byte, 1500)
			for {
				n, _, err := conn.ReadFrom(b)
				if err != nil {
					if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
						log.Error(err.Error())
					}
					return
				}
				m := make(map[string]interface{})
				err = json.Unmarshal(b[:n], &m)
				if err != nil {
					log.Error(err.Error())
					continue
				}
				message, err := messageSchema.Apply(m, s.FailOnRequired)
				if err != nil {
					log.Error(err.Error())
					continue
				}
				d.update(config, message)
			}
		}()
	}
	wg.Wait()
}

func (d *Discovery) update(config InterfaceConfig, message mapstr.M) {
	log := d.log

	v, err := message.GetValue("agent.id")
	if err != nil {
		log.Error("failed to update agent without id: ", err)
		return
	}
	agentID, ok := v.(string)
	if len(agentID) == 0 || !ok {
		log.Error("incorrect or empty agent id in Jolokia Discovery response")
		return
	}

	url, err := message.GetValue("url")
	if err != nil || url == nil {
		// This can happen if Jolokia agent is initializing and it still
		// doesn't know its URL
		log.Infof("agent %s without url, ignoring by now", agentID)
		return
	}

	d.Lock()
	defer d.Unlock()
	i, found := d.instances[agentID]
	if !found {
		i = &Instance{Message: message, AgentID: agentID}
		d.instances[agentID] = i
		d.events <- Event{d.ProviderUUID, "start", agentID, message}
	}
	i.LastSeen = time.Now()
	i.LastInterface = &config
}

func (d *Discovery) checkStopped() {
	d.Lock()
	defer d.Unlock()

	for id, i := range d.instances {
		if time.Since(i.LastSeen) > i.LastInterface.GracePeriod {
			d.events <- Event{d.ProviderUUID, "stop", i.AgentID, i.Message}
			delete(d.instances, id)
		}
	}
}
