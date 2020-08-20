// Package beatsouts allows the reuse of existing libbeat based outputs.
//
// TODO: The packag currently wraps libbeat outputs by accessing the outputs registry.
//       It would be better to allow developers to create wrappers more selectively, such that
//       configuration rewrites are possible.
package beatsout

//go:generate godocdown -plain=false -output Readme.md

import (
	"context"
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/idxmgmt"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	beatpipe "github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	"github.com/elastic/beats/v7/x-pack/club/internal/publishing"
	"github.com/elastic/go-concert/unison"
)

type beatsOutputFactory struct {
	info beat.Info
}

type beatsOutput struct {
	info            beat.Info
	pipeConfig      beatpipe.Config
	outputType      string
	indexManagement idxmgmt.Supporter
	cfg             *common.Config
}

type beatsPipeline struct {
	mu       unison.Mutex
	closeCtx context.Context
	cancelFn context.CancelFunc

	internal         *beatpipe.Pipeline
	clientPub        beat.Client
	clientDropIf     beat.Client
	clientGuaranteed beat.Client
	publishReturn    publishReturn
	ackHandler       publishing.ACKCallback
}

type publishReturn struct{ err error }

func (p *publishReturn) Closing()                      {}
func (p *publishReturn) Closed()                       {}
func (p *publishReturn) Published()                    { p.err = nil }
func (p *publishReturn) FilteredOut(_ beat.Event)      { p.err = errEventFlitered }
func (p *publishReturn) DroppedOnPublish(_ beat.Event) { p.err = errEventDropped }

var errEventFlitered = errors.New("event filtered out")
var errEventDropped = errors.New("event dropped")

// NewOutputFactory creates a new publishing.OutputFactory, that can be used to create outputs based
// on existing libbeat outputs.
//
// When creating an output we create a complete libbeat publisher pipeline
// including queue, ack handling and actual libbeat outputs for publishing the
// events. The pipeline is wrapped, such that is satifies the publishing.Output interface.
func NewOutputFactory(info beat.Info) publishing.OutputFactory {
	return &beatsOutputFactory{info: info}
}

func (f *beatsOutputFactory) ConfigureOutput(_ *logp.Logger, cfg *common.Config) (publishing.Output, error) {
	var pipeConfig beatpipe.Config
	if err := cfg.Unpack(&pipeConfig); err != nil {
		return nil, err
	}

	typeInfo := struct{ Type string }{}
	if err := cfg.Unpack(&typeInfo); err != nil {
		return nil, err
	}
	if outputs.FindFactory(typeInfo.Type) == nil {
		return nil, fmt.Errorf("unknown output type %v", typeInfo.Type)
	}

	// XXX: A little overkill to init all index management, but makes output setup easier for now
	indexManagementConfig := common.MustNewConfigFrom(map[string]interface{}{
		"setup.ilm.enabled":      false,
		"setup.template.enabled": false,
		"output.something":       map[string]interface{}{},
	})

	indexManagement, err := idxmgmt.MakeDefaultSupport(nil)(nil, f.info, indexManagementConfig)
	if err != nil {
		// the config is hard coded, if we panic here, we've messed up
		panic(err)
	}

	return &beatsOutput{
		info:            f.info,
		pipeConfig:      pipeConfig,
		outputType:      typeInfo.Type,
		indexManagement: indexManagement,
		cfg:             cfg,
	}, nil
}

func (f *beatsOutput) Open(_ unison.Canceler, log *logp.Logger, acks publishing.ACKCallback) (publishing.Publisher, error) {
	outputPipeline, err := beatpipe.Load(f.info,
		beatpipe.Monitors{
			Metrics:   nil,
			Telemetry: nil,
			Logger:    log.Named("publish"),
			Tracer:    nil,
		},
		f.pipeConfig,
		nil,
		makeOutputFactory(f.info, f.indexManagement, f.outputType, f.cfg),
	)
	if err != nil {
		return nil, fmt.Errorf("invalid output configuration: %w", err)
	}

	ctx, cancelFn := context.WithCancel(context.Background())

	pipe := &beatsPipeline{
		mu:         unison.MakeMutex(),
		internal:   outputPipeline,
		closeCtx:   ctx,
		cancelFn:   cancelFn,
		ackHandler: acks,
	}
	pipe.clientPub = connectOutputPipeline(ctx, outputPipeline, beat.OutputChooses, &pipe.publishReturn, acks)
	pipe.clientDropIf = connectOutputPipeline(ctx, outputPipeline, beat.DropIfFull, &pipe.publishReturn, acks)
	pipe.clientGuaranteed = connectOutputPipeline(ctx, outputPipeline, beat.GuaranteedSend, &pipe.publishReturn, acks)

	return pipe, nil
}

func makeOutputFactory(
	info beat.Info,
	indexManagement idxmgmt.Supporter,
	outputType string,
	cfg *common.Config,
) func(outputs.Observer) (string, outputs.Group, error) {
	return func(outStats outputs.Observer) (string, outputs.Group, error) {
		out, err := outputs.Load(indexManagement, info, outStats, outputType, cfg)
		return outputType, out, err
	}
}

func connectOutputPipeline(closeref beat.CloseRef, p beat.Pipeline, mode beat.PublishMode, events beat.ClientEventer, acks publishing.ACKCallback) beat.Client {
	var ackHandler beat.ACKer
	if mode != beat.DropIfFull {
		ackHandler = acker.EventPrivateReporter(func(_ int, data []interface{}) {
			for i := len(data) - 1; i != -1; i-- {
				id := data[i].(publishing.EventID)
				acks.UpdateEventStatus(id, publishing.EventPublished)
			}
		})
	}

	c, err := p.ConnectWith(beat.ClientConfig{
		PublishMode: mode,
		Events:      events,
		CloseRef:    closeref,
		ACKHandler:  ackHandler,
	})
	if err != nil {
		panic(err)
	}
	return c
}

func (out *beatsPipeline) Close() error {
	out.clientPub.Close()
	out.clientDropIf.Close()
	out.clientGuaranteed.Close()
	return out.internal.Close()
}

func (out *beatsPipeline) Publish(mode beat.PublishMode, id publishing.EventID, event beat.Event) error {
	var client beat.Client
	switch mode {
	case beat.GuaranteedSend:
		client = out.clientGuaranteed
	case beat.DropIfFull:
		client = out.clientDropIf
	default:
		client = out.clientPub
	}

	if err := out.mu.LockContext(out.closeCtx); err != nil {
		return err
	}
	defer out.mu.Unlock()

	out.publishReturn.err = nil
	event.Private = id

	// TODO: we need to be able to pass a cancelation context here in order to unblock the client
	client.Publish(event)
	err := out.publishReturn.err
	if err != nil {
		if cancelErr := out.closeCtx.Err(); cancelErr != nil {
			return cancelErr
		}
		return err
	}

	if mode == beat.DropIfFull {
		// XXX: we can not register an ACK handler if DropIfFull is set. Therefore we trigger the ACK
		//      right away.
		out.ackHandler.UpdateEventStatus(id, publishing.EventPending)
	}
	return nil
}
