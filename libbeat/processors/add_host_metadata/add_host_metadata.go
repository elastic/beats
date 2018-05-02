package add_host_metadata

import (
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/go-sysinfo"
	"github.com/elastic/go-sysinfo/types"
)

func init() {
	processors.RegisterPlugin("add_host_metadata", newHostMetadataProcessor)
}

type addHostMetadata struct {
	info       types.HostInfo
	lastUpdate time.Time
	data       common.MapStr
}

const (
	cacheExpiration = time.Minute * 5
)

func newHostMetadataProcessor(_ *common.Config) (processors.Processor, error) {
	h, err := sysinfo.Host()
	if err != nil {
		return nil, err
	}
	p := &addHostMetadata{
		info: h.Info(),
	}
	return p, nil
}

// Run enriches the given event with the host meta data
func (p *addHostMetadata) Run(event *beat.Event) (*beat.Event, error) {
	p.loadData()
	event.Fields.DeepUpdate(p.data)
	return event, nil
}

func (p *addHostMetadata) loadData() {

	// Check if cache is expired
	if p.lastUpdate.Add(cacheExpiration).Before(time.Now()) {
		p.data = common.MapStr{
			"host": common.MapStr{
				"name":         p.info.Hostname,
				"architecture": p.info.Architecture,
				"os": common.MapStr{
					"platform": p.info.OS.Platform,
					"version":  p.info.OS.Version,
					"family":   p.info.OS.Family,
				},
			},
		}

		// Optional params
		if p.info.UniqueID != "" {
			p.data.Put("host.id", p.info.UniqueID)
		}
		if p.info.Containerized != nil {
			p.data.Put("host.containerized", *p.info.Containerized)
		}
		if p.info.OS.Codename != "" {
			p.data.Put("host.os.codename", p.info.OS.Codename)
		}
		if p.info.OS.Build != "" {
			p.data.Put("host.os.build", p.info.OS.Build)
		}
		p.lastUpdate = time.Now()
	}
}

func (p addHostMetadata) String() string {
	return "add_host_metadata=[]"
}
