package publish

import (
	"errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/publisher/bc/publisher"
	"github.com/elastic/beats/libbeat/publisher/beat"
)

type TransactionPublisher struct {
	done      chan struct{}
	pipeline  publisher.Publisher
	canDrop   bool
	processor transProcessor
}

type transProcessor struct {
	ignoreOutgoing bool
	localIPs       []string
	name           string
}

var debugf = logp.MakeDebug("publish")

func NewTransactionPublisher(
	pipeline publisher.Publisher,
	ignoreOutgoing bool,
	canDrop bool,
) (*TransactionPublisher, error) {
	localIPs, err := common.LocalIPAddrsAsStrings(false)
	if err != nil {
		return nil, err
	}

	name := pipeline.(*publisher.BeatPublisher).GetName()
	p := &TransactionPublisher{
		done:     make(chan struct{}),
		pipeline: pipeline,
		canDrop:  canDrop,
		processor: transProcessor{
			localIPs:       localIPs,
			name:           name,
			ignoreOutgoing: ignoreOutgoing,
		},
	}
	return p, nil
}

func (p *TransactionPublisher) Stop() {
	close(p.done)
}

func (p *TransactionPublisher) CreateReporter(
	config *common.Config,
) (func(beat.Event), error) {

	// load and register the module it's fields, tags and processors settings
	meta := struct {
		Event      common.EventMetadata    `config:",inline"`
		Processors processors.PluginConfig `config:"processors"`
	}{}
	if err := config.Unpack(&meta); err != nil {
		return nil, err
	}

	processors, err := processors.New(meta.Processors)
	if err != nil {
		return nil, err
	}

	clientConfig := beat.ClientConfig{
		EventMetadata: meta.Event,
		Processor:     processors,
	}
	if p.canDrop {
		clientConfig.PublishMode = beat.DropIfFull
	}

	client, err := p.pipeline.ConnectX(clientConfig)
	if err != nil {
		return nil, err
	}

	// start worker, so post-processing and processor-pipeline
	// can work concurrently to sniffer acquiring new events
	ch := make(chan beat.Event, 3)
	go p.worker(ch, client)
	return func(event beat.Event) {
		ch <- event
	}, nil
}

func (p *TransactionPublisher) worker(ch chan beat.Event, client beat.Client) {
	go func() {
		<-p.done
		close(ch)
	}()

	for event := range ch {
		pub, _ := p.processor.Run(&event)
		if pub != nil {
			client.Publish(*pub)
		}
	}
}

func (p *transProcessor) Run(event *beat.Event) (*beat.Event, error) {
	if err := validateEvent(event); err != nil {
		logp.Warn("Dropping invalid event: %v", err)
		return nil, nil
	}

	if !p.normalizeTransAddr(event.Fields) {
		return nil, nil
	}

	return event, nil
}

// filterEvent validates an event for common required fields with types.
// If event is to be filtered out the reason is returned as error.
func validateEvent(event *beat.Event) error {
	fields := event.Fields

	if event.Timestamp.IsZero() {
		return errors.New("missing '@timestamp'")
	}

	_, ok := fields["@timestamp"]
	if ok {
		return errors.New("duplicate '@timestamp' field from event")
	}

	t, ok := fields["type"]
	if !ok {
		return errors.New("missing 'type' field from event")
	}

	_, ok = t.(string)
	if !ok {
		return errors.New("invalid 'type' field from event")
	}

	return nil
}

func (p *transProcessor) normalizeTransAddr(event common.MapStr) bool {
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

func (p *transProcessor) IsPublisherIP(ip string) bool {
	for _, myip := range p.localIPs {
		if myip == ip {
			return true
		}
	}
	return false
}

func (p *transProcessor) GetServerName(ip string) string {

	// in case the IP is localhost, return current shipper name
	islocal, err := common.IsLoopback(ip)
	if err != nil {
		logp.Err("Parsing IP %s fails with: %s", ip, err)
		return ""
	}

	if islocal {
		return p.name
	}

	return ""
}
