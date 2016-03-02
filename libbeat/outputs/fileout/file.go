package fileout

import (
	"encoding/json"

	"github.com/urso/ucfg"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
)

func init() {
	outputs.RegisterOutputPlugin("file", New)
}

type fileOutput struct {
	rotator logp.FileRotator
}

func New(cfg *ucfg.Config, _ int) (outputs.Outputer, error) {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	// disable bulk support in publisher pipeline
	cfg.SetInt("flush_interval", 0, -1)
	cfg.SetInt("bulk_max_size", 0, -1)

	output := &fileOutput{}
	if err := output.init(config); err != nil {
		return nil, err
	}
	return output, nil
}

func (out *fileOutput) init(config config) error {
	out.rotator.Path = config.Path
	out.rotator.Name = config.Filename
	if out.rotator.Name == "" {
		out.rotator.Name = config.Index
	}
	logp.Info("File output base filename set to: %v", out.rotator.Name)

	rotateeverybytes := uint64(config.RotateEveryKb) * 1024
	logp.Info("Rotate every bytes set to: %v", rotateeverybytes)
	out.rotator.RotateEveryBytes = &rotateeverybytes

	keepfiles := config.NumberOfFiles
	logp.Info("Number of files set to: %v", keepfiles)
	out.rotator.KeepFiles = &keepfiles

	err := out.rotator.CreateDirectory()
	if err != nil {
		return err
	}

	err = out.rotator.CheckIfConfigSane()
	if err != nil {
		return err
	}

	return nil
}

func (out *fileOutput) PublishEvent(
	trans outputs.Signaler,
	opts outputs.Options,
	event common.MapStr,
) error {
	jsonEvent, err := json.Marshal(event)
	if err != nil {
		// mark as success so event is not sent again.
		outputs.SignalCompleted(trans)

		logp.Err("Fail to convert the event to JSON: %s", err)
		return err
	}

	err = out.rotator.WriteLine(jsonEvent)
	if err != nil {
		if opts.Guaranteed {
			logp.Critical("Unable to write events to file: %s", err)
		} else {
			logp.Err("Error when writing line to file: %s", err)
		}
	}
	outputs.Signal(trans, err)
	return err
}
