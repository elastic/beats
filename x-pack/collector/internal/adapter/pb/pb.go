// pb wraps packetbeat so the sniffer and analyzers can be used as inputs.
//
// ## Input "sniffer" settings
//
// **interface.devices**: List of devices to collect packets from. An independent sniffer and set of network analyzers will be run per device.
//
// **interface.type**: Sniffer type. For example af_packet or pcap
//
// **interface.buffer_size_mb**:
//
// **interface.auto_promisc_mode** (NOT yet implemented): Put device into promisc mode.
//
// **interface.snaplen**:
//
// **interface.with_vlans**:
//
// **interface.bpf_filter**:
//
// **flows.X**: See packetbeat flows settings.
//
// **protocols.X**: See packetbeat protocol settings.
//
// **ignore_outgoing**:
//
package pb

//go:generate godocdown -plain=false -output Readme.md

import (
	"context"
	"errors"
	"fmt"
	"sync"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/cleanup"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/go-concert/ctxtool"
	"github.com/tsg/gopacket/layers"

	// import protocol modules

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/packetbeat/decoder"
	"github.com/elastic/beats/v7/packetbeat/flows"
	_ "github.com/elastic/beats/v7/packetbeat/include"
	"github.com/elastic/beats/v7/packetbeat/procs"
	"github.com/elastic/beats/v7/packetbeat/protos"
	"github.com/elastic/beats/v7/packetbeat/protos/icmp"
	"github.com/elastic/beats/v7/packetbeat/protos/tcp"
	"github.com/elastic/beats/v7/packetbeat/protos/udp"
	"github.com/elastic/beats/v7/packetbeat/publish"
	"github.com/elastic/beats/v7/packetbeat/sniffer"

	pbconfig "github.com/elastic/beats/v7/packetbeat/config"
)

type packetbeatInput struct {
	sniffers []*snifferInput
}

type snifferInput struct {
	// configuration
	device           string
	ifc              interfaceConfig
	ignoreOutgoing   bool
	analyzers        []*common.Config
	flowsConfig      *pbconfig.Flows
	icmpConfig       *common.Config
	internalNetworks []string

	// run state
	transPub     *publish.TransactionPublisher
	flows        *flows.Flows
	protocols    protos.ProtocolsStruct
	procsWatcher procs.ProcessesWatcher
}

type packetbeatConfig struct {
	Interface      devicesConfig
	Flows          *pbconfig.Flows
	Protocols      []*common.Config  `config:"protocols"`
	IgnoreOutgoing bool              `config:"ignore_outgoing"`
	Procs          procs.ProcsConfig `config:"procs"`
}

type devicesConfig struct {
	Devices   []string        `config:"devices"`
	Interface interfaceConfig `config:",inline"`
}

type interfaceConfig struct {
	Type                  string   `config:"type"`
	WithVlans             bool     `config:"with_vlans"`
	BpfFilter             string   `config:"bpf_filter"`
	Snaplen               int      `config:"snaplen"`
	BufferSizeMb          int      `config:"buffer_size_mb"`
	EnableAutoPromiscMode bool     `config:"auto_promisc_mode"`
	InternalNetworks      []string `config:"internal_networks"`
}

// Plugin provides a v2 input plugin implementation of packetbeat that allows
// packetbeat functionality as an input.
//
// The input name is "sniffer".
//
// Each input instance will be independent and can read from multiple devices.
// Multiple "sniffer" inputs can be run concurrently within a single process.
func Plugin() v2.Plugin {
	return v2.Plugin{
		Name:      "sniffer",
		Stability: feature.Experimental,
		Manager:   v2.ConfigureWith(configurePacketbeatInput),
	}
}

func (cfg *packetbeatConfig) Validate() error {
	if len(cfg.Interface.Devices) == 0 {
		return errors.New("no device configured")
	}
	if !cfg.Flows.IsEnabled() && len(cfg.Protocols) == 0 {
		return errors.New("no data collection configured, use flows or protocols settings")
	}

	return nil
}

func configurePacketbeatInput(cfg *common.Config) (v2.Input, error) {
	var pbcfg packetbeatConfig
	if err := cfg.Unpack(&pbcfg); err != nil {
		return nil, err
	}

	watcher := procs.ProcessesWatcher{}
	err := watcher.Init(pbcfg.Procs)
	if err != nil {
		return nil, err
	}

	var icmpConfig *common.Config
	var analyzers []*common.Config
	for _, config := range pbcfg.Protocols {
		if !config.Enabled() {
			continue
		}

		module := struct {
			Type string `config:"type" validate:"required"`
		}{}
		if err := config.Unpack(&module); err != nil {
			return nil, err
		}

		if module.Type == "icmp" {
			icmpConfig = config
			continue
		}

		proto := protos.Lookup(module.Type)
		if proto == protos.UnknownProtocol {
			return nil, fmt.Errorf("unkown protocol %v", module.Type)
		}

		analyzers = append(analyzers, config)
	}

	var inputs []*snifferInput
	for _, deviceName := range pbcfg.Interface.Devices {
		snifferInput := &snifferInput{
			device: deviceName,
			ifc:    pbcfg.Interface.Interface,

			ignoreOutgoing:   pbcfg.IgnoreOutgoing,
			analyzers:        analyzers,
			flowsConfig:      pbcfg.Flows,
			icmpConfig:       icmpConfig,
			internalNetworks: pbcfg.Interface.Interface.InternalNetworks,
			procsWatcher:     watcher,
		}

		inputs = append(inputs, snifferInput)
	}

	return &packetbeatInput{sniffers: inputs}, nil
}

func (p *packetbeatInput) Name() string                { return "sniffer" }
func (p *packetbeatInput) Test(_ v2.TestContext) error { return nil }
func (p *packetbeatInput) Run(ctx v2.Context, pipeline beat.PipelineConnector) error {
	var wg sync.WaitGroup
	for _, sniffer := range p.sniffers {
		sniffer := sniffer
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx.Logger = ctx.Logger.With("device", sniffer.device)
			err := sniffer.Run(ctx, pipeline)
			if err != nil && err != context.Canceled {
				ctx.Logger.Error("Sniffer failed with: %v", err)
			}
		}()
	}

	wg.Wait()
	return nil
}

func (p *snifferInput) Run(ctx v2.Context, pipeline beat.PipelineConnector) error {
	// setup protocol transaction analyzers and publishing
	var err error
	p.transPub, err = publish.NewTransactionPublisher(
		ctx.Agent.Name,
		pipeline,
		p.ignoreOutgoing,
		true,
		p.internalNetworks,
	)
	if err != nil {
		return err
	}
	defer p.transPub.Stop()

	err = p.protocols.Init(false, p.transPub, p.procsWatcher, nil, p.analyzers)
	if err != nil {
		return fmt.Errorf("Initializing protocol analyzers failed: %v", err)
	}

	filter := p.ifc.BpfFilter
	if filter == "" && !p.flowsConfig.IsEnabled() {
		filter = p.protocols.BpfFilter(p.ifc.WithVlans, p.icmpConfig.Enabled())
	}

	interfaceConfig := pbconfig.InterfacesConfig{
		Device:                p.device,
		Type:                  p.ifc.Type,
		WithVlans:             p.ifc.WithVlans,
		BpfFilter:             filter,
		Snaplen:               p.ifc.Snaplen,
		BufferSizeMb:          p.ifc.BufferSizeMb,
		EnableAutoPromiscMode: p.ifc.EnableAutoPromiscMode,
	}
	sniffer, err := sniffer.New(false, filter, p.createWorker, interfaceConfig)
	if err != nil {
		return err
	}
	_, cancel := ctxtool.WithFunc(ctxtool.FromCanceller(ctx.Cancelation), func() {
		sniffer.Stop()
	})
	defer cancel()

	// setup flows and flow data publishing
	flows, flowsClient, err := setupFlows(pipeline, p.procsWatcher, p.flowsConfig)
	if err != nil {
		return err
	}
	if flowsClient != nil {
		defer flowsClient.Close()
	}
	if flows != nil {
		flows.Start()
		defer flows.Stop()
	}

	p.flows = flows

	ctx.Logger.Info("Start sniffer")
	defer ctx.Logger.Info("Stop sniffer")
	return sniffer.Run()
}

func (p *snifferInput) createWorker(linkType layers.LinkType) (sniffer.Worker, error) {
	var icmp4 icmp.ICMPv4Processor
	var icmp6 icmp.ICMPv6Processor
	if p.icmpConfig.Enabled() {
		reporter, err := p.transPub.CreateReporter(p.icmpConfig)
		if err != nil {
			return nil, err
		}

		icmp, err := icmp.New(false, reporter, p.procsWatcher, p.icmpConfig)
		if err != nil {
			return nil, err
		}

		icmp4 = icmp
		icmp6 = icmp
	}

	// NOTE: tcp and udp start background processes that might not be cleaned up correctly
	tcp, err := tcp.NewTCP(&p.protocols)
	if err != nil {
		return nil, err
	}

	udp, err := udp.NewUDP(&p.protocols)
	if err != nil {
		return nil, err
	}

	worker, err := decoder.New(p.flows, linkType, icmp4, icmp6, tcp, udp)
	if err != nil {
		return nil, err
	}
	return worker, nil
}

func setupFlows(pipeline beat.PipelineConnector, watcher procs.ProcessesWatcher, config *pbconfig.Flows) (*flows.Flows, beat.Client, error) {
	if !config.IsEnabled() {
		return nil, nil, nil
	}

	processors, err := processors.New(config.Processors)
	if err != nil {
		return nil, nil, err
	}

	client, err := pipeline.ConnectWith(beat.ClientConfig{
		Processing: beat.ProcessingConfig{
			EventMetadata: config.EventMetadata,
			Processor:     processors,
			KeepNull:      config.KeepNull,
		},
	})
	if err != nil {
		return nil, nil, err
	}
	ok := false
	defer cleanup.IfNot(&ok, func() { client.Close() })

	flows, err := flows.NewFlows(client.PublishAll, watcher, config)
	if err != nil {
		return nil, nil, err
	}

	ok = true
	return flows, client, nil
}
