package pipeline

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
	"github.com/elastic/go-concert/unison"
)

type outputFactory interface {
	ConfigureOutput(*logp.Logger, *common.Config, outputACKCallback) (output, error)
}

type output interface {
	Close() error
	Publish(beat.PublishMode, eventID, beat.Event) error
}

type outputACKCallback interface {
	EventSendStatus(eventID, eventStatus)
}

type beatsPipelineOutput struct {
	mu       unison.Mutex
	closeCtx context.Context
	cancelFn context.CancelFunc

	internal         *beatpipe.Pipeline
	clientPub        beat.Client
	clientDropIf     beat.Client
	clientGuaranteed beat.Client
	publishReturn    publishReturn
	ackHandler       outputACKCallback
}

type eventID uint64

type eventStatus uint8

const (
	eventPublished eventStatus = iota
	eventPending
	eventInvalid // invalid events can not be published
	eventFailed  // event could not be send and is finally dropped by the output
)

type publishReturn struct{ err error }

func (p *publishReturn) Closing()                      {}
func (p *publishReturn) Closed()                       {}
func (p *publishReturn) Published()                    { p.err = nil }
func (p *publishReturn) FilteredOut(_ beat.Event)      { p.err = errEventFlitered }
func (p *publishReturn) DroppedOnPublish(_ beat.Event) { p.err = errEventDropped }

var errEventFlitered = errors.New("event filtered out")
var errEventDropped = errors.New("event dropped")

func createOutput(log *logp.Logger, info beat.Info, cfg *common.Config, acks outputACKCallback) (*beatsPipelineOutput, error) {
	var pipeConfig beatpipe.Config
	if err := cfg.Unpack(&pipeConfig); err != nil {
		return nil, err
	}

	typeInfo := struct{ Type string }{}
	if err := cfg.Unpack(&typeInfo); err != nil {
		return nil, err
	}

	// XXX: A little overkill to init all index management, but makes output setup easier for now
	indexManagementConfig := common.MustNewConfigFrom(map[string]interface{}{
		"setup.ilm.enabled":      false,
		"setup.template.enabled": false,
		"output.something":       map[string]interface{}{},
	})

	indexManagement, err := idxmgmt.MakeDefaultSupport(nil)(nil, info, indexManagementConfig)
	if err != nil {
		// the config is hard coded, if we panic here, we've messed up
		panic(err)
	}

	outputPipeline, err := beatpipe.Load(info,
		beatpipe.Monitors{
			Metrics:   nil,
			Telemetry: nil,
			Logger:    log.Named("publish"),
			Tracer:    nil,
		},
		pipeConfig,
		nil,
		makeOutputFactory(info, indexManagement, typeInfo.Type, cfg),
	)
	if err != nil {
		return nil, fmt.Errorf("invalid output configuration: %w", err)
	}

	ctx, cancelFn := context.WithCancel(context.Background())

	out := &beatsPipelineOutput{
		mu:         unison.MakeMutex(),
		internal:   outputPipeline,
		closeCtx:   ctx,
		cancelFn:   cancelFn,
		ackHandler: acks,
	}
	out.clientPub = connectOutputPipeline(ctx, outputPipeline, beat.OutputChooses, &out.publishReturn, acks)
	out.clientDropIf = connectOutputPipeline(ctx, outputPipeline, beat.DropIfFull, &out.publishReturn, acks)
	out.clientGuaranteed = connectOutputPipeline(ctx, outputPipeline, beat.GuaranteedSend, &out.publishReturn, acks)

	return out, nil
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

func connectOutputPipeline(closeref beat.CloseRef, p beat.Pipeline, mode beat.PublishMode, events beat.ClientEventer, acks outputACKCallback) beat.Client {
	var ackHandler beat.ACKer
	if mode != beat.DropIfFull {
		ackHandler = acker.EventPrivateReporter(func(_ int, data []interface{}) {
			for i := len(data) - 1; i != -1; i-- {
				id := data[i].(eventID)
				acks.EventSendStatus(id, eventPublished)
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

func (out *beatsPipelineOutput) Close() error {
	out.clientPub.Close()
	out.clientDropIf.Close()
	out.clientGuaranteed.Close()
	return out.internal.Close()
}

func (out *beatsPipelineOutput) Publish(mode beat.PublishMode, id eventID, event beat.Event) error {
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
		out.ackHandler.EventSendStatus(id, eventPending)
	}
	return nil
}
