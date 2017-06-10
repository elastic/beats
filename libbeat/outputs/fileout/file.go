package fileout

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/codec"
	"github.com/elastic/beats/libbeat/publisher"
)

func init() {
	outputs.RegisterType("file", makeFileout)
}

type fileOutput struct {
	beat    common.BeatInfo
	rotator logp.FileRotator
	codec   codec.Codec
}

// New instantiates a new file output instance.
func makeFileout(beat common.BeatInfo, cfg *common.Config) (outputs.Group, error) {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return outputs.Fail(err)
	}

	// disable bulk support in publisher pipeline
	cfg.SetInt("flush_interval", -1, -1)
	cfg.SetInt("bulk_max_size", -1, -1)

	fo := &fileOutput{beat: beat}
	if err := fo.init(config); err != nil {
		return outputs.Fail(err)
	}

	return outputs.Success(-1, 0, fo)
}

func (out *fileOutput) init(config config) error {
	var err error

	out.rotator.Path = config.Path
	out.rotator.Name = config.Filename
	if out.rotator.Name == "" {
		out.rotator.Name = out.beat.Beat
	}

	enc, err := codec.CreateEncoder(config.Codec)
	if err != nil {
		return err
	}

	out.codec = enc

	logp.Info("File output path set to: %v", out.rotator.Path)
	logp.Info("File output base filename set to: %v", out.rotator.Name)

	rotateeverybytes := uint64(config.RotateEveryKb) * 1024
	logp.Info("Rotate every bytes set to: %v", rotateeverybytes)
	out.rotator.RotateEveryBytes = &rotateeverybytes

	keepfiles := config.NumberOfFiles
	logp.Info("Number of files set to: %v", keepfiles)
	out.rotator.KeepFiles = &keepfiles

	err = out.rotator.CreateDirectory()
	if err != nil {
		return err
	}

	err = out.rotator.CheckIfConfigSane()
	if err != nil {
		return err
	}

	return nil
}

// Implement Outputer
func (out *fileOutput) Close() error {
	return nil
}

func (out *fileOutput) Publish(
	batch publisher.Batch,
) error {
	defer batch.ACK()

	events := batch.Events()
	for i := range events {
		event := &events[i]

		serializedEvent, err := out.codec.Encode(out.beat.Beat, &event.Content)
		if err != nil {
			if event.Guaranteed() {
				logp.Critical("Failed to serialize the event: %v", err)
			} else {
				logp.Warn("Failed to serialize the event: %v", err)
			}
			continue
		}

		err = out.rotator.WriteLine(serializedEvent)
		if err != nil {
			if event.Guaranteed() {
				logp.Critical("Writing event to file failed with: %v", err)
			} else {
				logp.Warn("Writing event to file failed with: %v", err)
			}
			continue
		}
	}

	return nil
}
