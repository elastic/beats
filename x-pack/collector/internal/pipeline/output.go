package pipeline

import (
	"context"
	"io"
	"sync/atomic"
	"unsafe"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/collector/internal/publishing"
)

type output struct {
	configHash string
	output     publishing.Output
}

type pipelineOutput struct {
	output    output
	id        uint64
	acker     *outputACKer
	publisher publishing.Publisher
}

type replacableOutput struct {
	ptr unsafe.Pointer
}

func newPipelineOutput(log *logp.Logger, outputID uint64, output output, events *eventTracker) (*pipelineOutput, error) {
	outputACKer := &outputACKer{id: outputID, events: events}
	publisher, err := output.output.Open(context.Background(), log, outputACKer)
	if err != nil {
		return nil, err
	}

	return &pipelineOutput{
		output:    output,
		id:        outputID,
		acker:     outputACKer,
		publisher: publisher,
	}, nil
}

func (o *pipelineOutput) Close() error {
	if o.publisher == nil {
		return nil
	}
	return o.publisher.Close()
}

func (o *pipelineOutput) Publish(mode beat.PublishMode, eventID publishing.EventID, event beat.Event) error {
	return o.publisher.Publish(mode, eventID, event)
}

func newReplacablePublisher(out *pipelineOutput) *replacableOutput {
	p := &replacableOutput{}
	p.SetActive(out)
	return p
}

func (o *replacableOutput) SetActive(out *pipelineOutput) *pipelineOutput {
	ptrOld := atomic.SwapPointer(&o.ptr, unsafe.Pointer(out))
	return (*pipelineOutput)(ptrOld)
}

func (o *replacableOutput) GetActive() *pipelineOutput {
	ptr := atomic.LoadPointer(&o.ptr)
	return (*pipelineOutput)(ptr)
}

func (r *replacableOutput) Close() error {
	out := r.SetActive(nil)
	if out != nil {
		return out.Close()
	}
	return nil
}

func (r *replacableOutput) Publish(mode beat.PublishMode, eventID publishing.EventID, event beat.Event) error {
	out := r.GetActive()
	if out == nil {
		return io.EOF
	}

	return out.Publish(mode, eventID, event)
}
