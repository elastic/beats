// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package netflow

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	filestream "github.com/elastic/beats/v7/filebeat/input/filestream"
	"github.com/elastic/beats/v7/filebeat/input/netmetrics"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/filebeat/inputsource"
	"github.com/elastic/beats/v7/filebeat/inputsource/udp"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/convert"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/fields"
	ipfix_reader "github.com/elastic/beats/v7/x-pack/libbeat/reader/ipfix"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/unison"
)

const (
	inputName = "netflow"
)

func Plugin(log *logp.Logger) v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "collect and decode packets of netflow protocol",
		Manager: &netflowInputManager{
			log: log.Named(inputName),
		},
	}
}

type netflowInputManager struct {
	log *logp.Logger
}

func (im *netflowInputManager) Init(_ unison.Group) error {
	return nil
}

func (im *netflowInputManager) Create(cfg *conf.C) (v2.Input, error) {
	inputCfg := defaultConfig
	if err := cfg.Unpack(&inputCfg); err != nil {
		return nil, err
	}

	customFields := make([]fields.FieldDict, len(inputCfg.CustomDefinitions))
	for idx, yamlPath := range inputCfg.CustomDefinitions {
		f, err := decoder.LoadFieldDefinitionsFromFile(yamlPath)
		if err != nil {
			return nil, fmt.Errorf("failed parsing custom field definitions from file '%s': %w", yamlPath, err)
		}
		customFields[idx] = f
	}

	input := &netflowInput{
		cfg:              inputCfg,
		customFields:     customFields,
		internalNetworks: inputCfg.InternalNetworks,
		logger:           im.log,
		queueSize:        inputCfg.PacketQueueSize,
	}

	return input, nil
}

type packet struct {
	data   []byte
	source net.Addr
}

type netflowInput struct {
	mtx              sync.Mutex
	cfg              config
	decoder          *decoder.Decoder
	filewatcher      *filestream.FSWatcher
	clients          []beat.Client
	metrics          *netflowMetrics
	udpMetrics       *netmetrics.UDP
	customFields     []fields.FieldDict
	internalNetworks []string
	logger           *logp.Logger
	queueC           chan packet
	wg               sync.WaitGroup
	ctx              context.Context
	cancelFunc       context.CancelFunc
	queueSize        int
	started          bool
}

func (n *netflowInput) Name() string {
	return inputName
}

func (n *netflowInput) Test(_ v2.TestContext) error {
	return nil
}

func (n *netflowInput) Run(env v2.Context, connector beat.PipelineConnector) error {
	n.mtx.Lock()
	if n.started {
		n.mtx.Unlock()
		return nil
	}
	n.ctx, n.cancelFunc = context.WithCancel(v2.GoContextFromCanceler(env.Cancelation))
	n.started = true
	n.mtx.Unlock()

	env.UpdateStatus(status.Starting, "Starting netflow input")
	n.logger.Info("Starting netflow input")

	n.logger.Info("Connecting to beat event publishing")

	const pollInterval = time.Minute
	n.udpMetrics = netmetrics.NewUDP("netflow", env.ID, n.cfg.Host, uint64(n.cfg.ReadBuffer), pollInterval, n.logger)
	defer n.udpMetrics.Close()

	n.metrics = newInputMetrics(n.udpMetrics.Registry())
	var err error

	// check for file paths -- only supporting one or the other
	if len(n.cfg.Ipfix.Paths) > 0 {
		err = n.setupFile(env, connector)
		if err != nil {
			return err
		}
	} else {
		err = n.setupOrig(env, connector)
		if err != nil {
			return err
		}
		n.logger.Info("Starting udp server")

		udpServer := udp.New(&n.cfg.Config, func(data []byte, metadata inputsource.NetworkMetadata) {
			if n.ctx.Err() != nil {
				return
			}
			select {
			case <-n.ctx.Done():
				return
			case n.queueC <- packet{data, metadata.RemoteAddr}:
			default:
				if discardedEvents := n.metrics.DiscardedEvents(); discardedEvents != nil {
					discardedEvents.Inc()
				}
			}
		})
		err = udpServer.Start()
		if err != nil {
			errorMsg := fmt.Sprintf("Failed to start udp server: %v", err)
			n.logger.Errorf(errorMsg)
			env.UpdateStatus(status.Failed, errorMsg)
			n.stop()
			return err
		}
		defer udpServer.Stop()

	}
	env.UpdateStatus(status.Running, "")
	<-n.ctx.Done()
	n.stop()

	return nil
}

func (n *netflowInput) setupFile(env v2.Context, connector beat.PipelineConnector) error {
	var err error
	n.decoder, err = decoder.NewDecoder(decoder.NewConfig(n.logger).
		WithProtocols(n.cfg.Protocols...).
		WithExpiration(n.cfg.ExpirationTimeout).
		WithCustomFields(n.customFields...).
		WithSequenceResetEnabled(n.cfg.DetectSequenceReset).
		WithSharedTemplates(n.cfg.ShareTemplates).
		WithActiveSessionsMetric(n.metrics.ActiveSessions()).
		WithCache(n.cfg.NumberOfWorkers > 1))
	if err != nil {
		env.UpdateStatus(status.Failed, fmt.Sprintf("Failed to initialize netflow decoder: %v", err))
		return fmt.Errorf("error initializing netflow decoder: %w", err)
	}

	// Start FSWatcher for n.cfg.Ipfix.Paths
	// Construct FSWatcher using the public filestream export
	watcher, err := filestream.NewFSWatcher(n.cfg.Ipfix.Paths, n.logger)
	if err != nil {
		env.UpdateStatus(status.Failed, fmt.Sprintf("Failed to start FSWatcher: %v", err))
		return fmt.Errorf("error starting FSWatcher: %w", err)
	}
	n.filewatcher = &watcher

	// Start watcher goroutine
	n.wg.Add(1)
	go func() {
		defer n.wg.Done()
		watcher.Run(n.ctx)
	}()

	client, err := connector.ConnectWith(beat.ClientConfig{
		PublishMode: beat.DefaultGuarantees,
		Processing: beat.ProcessingConfig{
			EventNormalization: boolPtr(false),
		},
		EventListener: nil,
	})
	if err != nil {
		env.UpdateStatus(status.Failed, fmt.Sprintf("Failed connecting to beat event publishing: %v", err))
		n.logger.Errorw("Failed connecting to beat event publishing", "error", err)
		n.stop()
		return err
	}

	// Start event handler goroutine
	n.wg.Add(1)
	n.clients = append(n.clients, client)
	go func(client beat.Client) {
		defer n.wg.Done()
		for {
			if n.ctx.Err() != nil {
				return
			}
			event := watcher.Event()
			if event.Op == filestream.OpCreate || event.Op == filestream.OpWrite {
				filename := event.Descriptor.Filename
				// TODO: Process file as Netflow/IPFIX
				n.logger.Infof("Detected new or changed file: %s", filename)
				n.processFile(filename, client)
			}
		}
	}(client)

	return nil
}

func (n *netflowInput) processFile(fpath string, client beat.Client) {
	if fpath == "" {
		return
	}
	n.logger.Infof("processing file [%v] now", fpath)
	start := time.Now().In(time.UTC)
	n.metrics.ipfix.FilesOpened.Inc()
	defer n.metrics.ipfix.FilesClosed.Inc()

	// this will actually be the file to read, not the packet
	fi, err := os.Stat(fpath)
	if err != nil {
		// log something
		n.logger.Warnf("Error stat on file [%s]: %v", fpath, err)
		return
	}

	// check for pipe?
	if fi.Mode()&os.ModeNamedPipe != 0 {
		n.logger.Warnf("Error on file %s: Named Pipes are not supported", fpath)
		return
	}
	// check for regular file?

	f, err := file.ReadOpen(fpath)
	if err != nil {
		n.logger.Warnf("Error ReadOpen on file %s: %v", fpath, err)
		return
	}

	defer f.Close()
	defer os.Remove(fpath)

	reader, err := n.addGzipDecoderIfNeeded(f)
	if err != nil {
		n.logger.Warnf("Failed to add gzip decoder: [%v]", err)
	}

	ip := ipfix_reader.Config{}
	decoder, err := ipfix_reader.NewBufferedReader(reader, &ip)
	for {
		if !decoder.Next() {
			break
		}
		events, err := decoder.Record()
		if err != nil {
			n.logger.Warnf("Error parsing NetFlow Record")
			if decodeErrors := n.metrics.DecodeErrors(); decodeErrors != nil {
				decodeErrors.Inc()
			}
			continue
		}

		client.PublishAll(events)
	}

	n.metrics.ipfix.ProcessingTime.Update(time.Since(start).Nanoseconds())
}

// Copied from x-pack/filebeat/input/awss3/s3_objects.go
//
// isStreamGzipped determines whether the given stream of bytes (encapsulated in a buffered reader)
// represents gzipped content or not. A buffered reader is used so the function can peek into the byte
// stream without consuming it. This makes it convenient for code executed after this function call
// to consume the stream if it wants.
func isStreamGzipped(r *bufio.Reader) (bool, error) {
	buf, err := r.Peek(3)
	if err != nil && err != io.EOF {
		return false, err
	}

	// gzip magic number (1f 8b) and the compression method (08 for DEFLATE).
	return bytes.HasPrefix(buf, []byte{0x1F, 0x8B, 0x08}), nil
}

func (p *netflowInput) addGzipDecoderIfNeeded(body io.Reader) (io.Reader, error) {
	bufReader := bufio.NewReader(body)

	gzipped, err := isStreamGzipped(bufReader)
	if err != nil {
		return nil, err
	}
	if !gzipped {
		return bufReader, nil
	}

	return gzip.NewReader(bufReader)
}

func (n *netflowInput) setupOrig(env v2.Context, connector beat.PipelineConnector) error {
	var err error
	n.decoder, err = decoder.NewDecoder(decoder.NewConfig(n.logger).
		WithProtocols(n.cfg.Protocols...).
		WithExpiration(n.cfg.ExpirationTimeout).
		WithCustomFields(n.customFields...).
		WithSequenceResetEnabled(n.cfg.DetectSequenceReset).
		WithSharedTemplates(n.cfg.ShareTemplates).
		WithActiveSessionsMetric(n.metrics.ActiveSessions()).
		WithCache(n.cfg.NumberOfWorkers > 1))
	if err != nil {
		env.UpdateStatus(status.Failed, fmt.Sprintf("Failed to initialize netflow decoder: %v", err))
		return fmt.Errorf("error initializing netflow decoder: %w", err)
	}

	n.logger.Info("Starting netflow decoder")
	if err := n.decoder.Start(); err != nil {
		env.UpdateStatus(status.Failed, fmt.Sprintf("Failed to start netflow decoder: %v", err))
		n.logger.Errorw("Failed to start netflow decoder", "error", err)
		n.stop()
		return err
	}

	n.queueC = make(chan packet, n.queueSize)
	for i := uint32(0); i < n.cfg.NumberOfWorkers; i++ {
		client, err := connector.ConnectWith(beat.ClientConfig{
			PublishMode: beat.DefaultGuarantees,
			Processing: beat.ProcessingConfig{
				EventNormalization: boolPtr(false),
			},
			EventListener: nil,
		})
		if err != nil {
			env.UpdateStatus(status.Failed, fmt.Sprintf("Failed connecting to beat event publishing: %v", err))
			n.logger.Errorw("Failed connecting to beat event publishing", "error", err)
			n.stop()
			return err
		}

		n.clients = append(n.clients, client)
		n.wg.Add(1)
		go func(client beat.Client) {
			defer n.wg.Done()
			for {
				select {
				case <-n.ctx.Done():
					return
				case pkt := <-n.queueC:
					pktStartTime := time.Now()
					flows, err := n.decoder.Read(bytes.NewBuffer(pkt.data), pkt.source)
					if err != nil {
						n.logger.Warnf("Error parsing NetFlow packet of length %d from %s: %v", len(pkt.data), pkt.source, err)
						if decodeErrors := n.metrics.DecodeErrors(); decodeErrors != nil {
							decodeErrors.Inc()
						}
						continue
					}

					fLen := len(flows)
					if fLen != 0 {
						evs := make([]beat.Event, fLen)
						if flowsTotal := n.metrics.Flows(); flowsTotal != nil {
							flowsTotal.Add(uint64(fLen))
						}
						for flowIdx, flow := range flows {
							evs[flowIdx] = convert.RecordToBeatEvent(flow, n.internalNetworks)
						}
						client.PublishAll(evs)
					}
					n.udpMetrics.Log(pkt.data, pktStartTime)
				}
			}
		}(client)
	}

	n.logger.Info("Starting udp server")

	udpServer := udp.New(&n.cfg.Config, func(data []byte, metadata inputsource.NetworkMetadata) {
		if n.ctx.Err() != nil {
			return
		}
		select {
		case <-n.ctx.Done():
			return
		case n.queueC <- packet{data, metadata.RemoteAddr}:
		default:
			if discardedEvents := n.metrics.DiscardedEvents(); discardedEvents != nil {
				discardedEvents.Inc()
			}
		}
	})
	err = udpServer.Start()
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to start udp server: %v", err)
		n.logger.Errorf(errorMsg)
		env.UpdateStatus(status.Failed, errorMsg)
		n.stop()
		return err
	}
	defer udpServer.Stop()

	env.UpdateStatus(status.Running, "")
	<-n.ctx.Done()
	n.stop()

	return nil
}

// stop stops the netflow input
func (n *netflowInput) stop() {
	n.mtx.Lock()
	defer n.mtx.Unlock()

	n.cancelFunc()
	n.wg.Wait()

	if !n.started {
		return
	}

	if n.decoder != nil {
		if err := n.decoder.Stop(); err != nil {
			n.logger.Errorw("Error stopping decoder", "error", err)
		}
		n.decoder = nil
	}

	if n.clients != nil {
		for _, client := range n.clients {
			if err := client.Close(); err != nil {
				n.logger.Errorw("Error closing beat client", "error", err)
			}
		}
	}

	close(n.queueC)

	n.started = false
}

func boolPtr(b bool) *bool { return &b }
