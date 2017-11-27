package fileout

import (
	"os"
	"path/filepath"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/file"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/codec"
	"github.com/elastic/beats/libbeat/publisher"
)

func init() {
	outputs.RegisterType("file", makeFileout)
}

type fileOutput struct {
	beat     beat.Info
	observer outputs.Observer
	rotator  *file.FileRotator
	codec    codec.Codec
	logger   *logp.Logger
}

// NewLogger instantiates a new file output instance.
func makeFileout(
	beat beat.Info,
	observer outputs.Observer,
	cfg *common.Config,
) (outputs.Group, error) {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return outputs.Fail(err)
	}

	// disable bulk support in publisher pipeline
	cfg.SetInt("bulk_max_size", -1, -1)

	fo := &fileOutput{
		beat:     beat,
		observer: observer,
		logger:   logp.NewLogger("output.file"),
	}
	if err := fo.init(beat, config); err != nil {
		return outputs.Fail(err)
	}

	return outputs.Success(-1, 0, fo)
}

func (out *fileOutput) init(beat beat.Info, c config) error {
	var path string
	if c.Filename != "" {
		path = filepath.Join(c.Path, c.Filename)
	} else {
		path = filepath.Join(c.Path, out.beat.Beat+".json")
	}

	var err error
	out.rotator, err = file.NewFileRotator(
		path,
		file.MaxSizeBytes(c.RotateEveryKb*1024),
		file.MaxBackups(c.NumberOfFiles),
		file.Permissions(os.FileMode(c.Permissions)),
	)
	if err != nil {
		return err
	}

	out.codec, err = codec.CreateEncoder(beat, c.Codec)
	if err != nil {
		return err
	}

	out.logger.Info("Initialized file output",
		logp.String("path", path),
		logp.Uint("max_size_bytes", c.RotateEveryKb*1024),
		logp.Uint("max_backups", c.NumberOfFiles),
		logp.Stringer("permissions", os.FileMode(c.Permissions)),
	)

	return nil
}

// Implement Outputer
func (out *fileOutput) Close() error {
	return out.rotator.Close()
}

func (out *fileOutput) Publish(
	batch publisher.Batch,
) error {
	defer batch.ACK()

	st := out.observer
	events := batch.Events()
	st.NewBatch(len(events))

	dropped := 0
	for i := range events {
		event := &events[i]

		serializedEvent, err := out.codec.Encode(out.beat.Beat, &event.Content)
		if err != nil {
			if event.Guaranteed() {
				out.logger.Error("Failed to serialize the event", logp.Error(err))
			} else {
				out.logger.Warn("Failed to serialize the event", logp.Error(err))
			}

			dropped++
			continue
		}

		if _, err = out.rotator.Write(append(serializedEvent, '\n')); err != nil {
			st.WriteError(err)

			if event.Guaranteed() {
				out.logger.Error("Writing event to file failed", logp.Error(err))
			} else {
				out.logger.Warn("Writing event to file failed", logp.Error(err))
			}

			dropped++
			continue
		}

		st.WriteBytes(len(serializedEvent) + 1)
	}

	st.Dropped(dropped)
	st.Acked(len(events) - dropped)

	return nil
}
