package main

import (
	// import protocol modules
	"errors"
	"fmt"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cleanup"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/packetbeat/decoder"
	"github.com/elastic/beats/v7/packetbeat/flows"
	_ "github.com/elastic/beats/v7/packetbeat/include"
	"github.com/elastic/beats/v7/packetbeat/protos"
	"github.com/elastic/beats/v7/packetbeat/protos/icmp"
	"github.com/elastic/beats/v7/packetbeat/protos/tcp"
	"github.com/elastic/beats/v7/packetbeat/protos/udp"
	"github.com/elastic/beats/v7/packetbeat/publish"
	"github.com/elastic/beats/v7/packetbeat/sniffer"
	"github.com/elastic/go-concert/ctxtool"
	"github.com/elastic/go-concert/unison"
	"github.com/tsg/gopacket/layers"

	pbconfig "github.com/elastic/beats/v7/packetbeat/config"
)

type packetbeatRegistry struct{}

type packetbeatManager struct{}

type packetbeatInput struct {
	config    packetbeatConfig
	sniffer   *sniffer.Sniffer
	analyzers []*common.Config

	// internal setup created during run (fields required to pass configured collection to createWorker)
	transPub   *publish.TransactionPublisher
	flows      *flows.Flows
	icmpConfig *common.Config
	protocols  protos.ProtocolsStruct
}

type packetbeatAnalyzer struct {
	protocol   string
	protocolID protos.Protocol
	plugin     protos.ProtocolPlugin
	config     *common.Config
}

type packetbeatConfig struct {
	Interface      pbconfig.InterfacesConfig
	Flows          *pbconfig.Flows
	Protocols      []*common.Config `config:"protocols"`
	IgnoreOutgoing bool             `config:"ignore_outgoing"`
}

func (cfg *packetbeatConfig) Validate() error {
	if cfg.Interface.Device == "" {
		return errors.New("no device configured")
	}
	if !cfg.Flows.IsEnabled() && len(cfg.Protocols) == 0 {
		return errors.New("no data collection configured, use flows or protocols settings")
	}

	return nil
}

func makePacketbeatRegistry() v2.Registry {
	return &packetbeatRegistry{}
}

func (p *packetbeatRegistry) Init(_ unison.Group, _ v2.Mode) error {
	return nil
}

func (p *packetbeatRegistry) Find(name string) (v2.Plugin, bool) {
	if name != "sniffer" {
		return v2.Plugin{}, false
	}
	return v2.Plugin{
		Name:      "sniffer",
		Stability: feature.Experimental,
		Manager:   &packetbeatManager{},
	}, true
}

func (p *packetbeatManager) Init(grp unison.Group, mode v2.Mode) error {
	return nil
}

func (p *packetbeatManager) Create(cfg *common.Config) (v2.Input, error) {
	var pbcfg packetbeatConfig
	if err := cfg.Unpack(&pbcfg); err != nil {
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

		proto := protos.Lookup(module.Type)
		if proto == protos.UnknownProtocol {
			return nil, fmt.Errorf("unkown protocol %v", module.Type)
		}

		if module.Type == "icmp" {
			icmpConfig = config
		}

		analyzers = append(analyzers, config)
	}

	withVlans := pbcfg.Interface.WithVlans
	withICMP := icmpConfig.Enabled()

	filter := pbcfg.Interface.BpfFilter
	if filter == "" && !pbcfg.Flows.IsEnabled() {
		filter = protos.Protos.BpfFilter(withVlans, withICMP)
	}

	var err error
	input := &packetbeatInput{
		config:     pbcfg,
		analyzers:  analyzers,
		icmpConfig: icmpConfig,
	}
	input.sniffer, err = sniffer.New(false, filter, input.createWorker, pbcfg.Interface)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize sniffer: %w", err)
	}

	return input, nil
}

func (p *packetbeatInput) Name() string                { return "sniffer" }
func (p *packetbeatInput) Test(_ v2.TestContext) error { return nil }
func (p *packetbeatInput) Run(ctx v2.Context, pipeline beat.PipelineConnector) error {
	_, cancel := ctxtool.WithFunc(ctxtool.FromCanceller(ctx.Cancelation), func() {
		p.sniffer.Stop()
	})
	defer cancel()

	// setup protocol transaction analyzers and publishing
	var err error
	p.transPub, err = publish.NewTransactionPublisher(
		ctx.Agent.Name,
		pipeline,
		p.config.IgnoreOutgoing,
		p.config.Interface.File == "",
	)
	if err != nil {
		return err
	}
	defer p.transPub.Stop()

	err = p.protocols.Init(false, p.transPub, nil, p.analyzers)
	if err != nil {
		return fmt.Errorf("Initializing protocol analyzers failed: %v", err)
	}

	// setup flows and flow data publishing
	flows, flowsClient, err := setupFlows(pipeline, p.config)
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
	return p.sniffer.Run()
}

func (p *packetbeatInput) createWorker(linkType layers.LinkType) (sniffer.Worker, error) {
	var icmp4 icmp.ICMPv4Processor
	var icmp6 icmp.ICMPv6Processor
	if p.icmpConfig.Enabled() {
		reporter, err := p.transPub.CreateReporter(p.icmpConfig)
		if err != nil {
			return nil, err
		}

		icmp, err := icmp.New(false, reporter, p.icmpConfig)
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

func setupFlows(pipeline beat.PipelineConnector, config packetbeatConfig) (*flows.Flows, beat.Client, error) {
	if !config.Flows.IsEnabled() {
		return nil, nil, nil
	}

	processors, err := processors.New(config.Flows.Processors)
	if err != nil {
		return nil, nil, err
	}

	client, err := pipeline.ConnectWith(beat.ClientConfig{
		Processing: beat.ProcessingConfig{
			EventMetadata: config.Flows.EventMetadata,
			Processor:     processors,
			KeepNull:      config.Flows.KeepNull,
		},
	})
	if err != nil {
		return nil, nil, err
	}
	ok := false
	defer cleanup.IfNot(&ok, func() { client.Close() })

	flows, err := flows.NewFlows(client.PublishAll, config.Flows)
	if err != nil {
		return nil, nil, err
	}

	ok = true
	return flows, client, nil
}
