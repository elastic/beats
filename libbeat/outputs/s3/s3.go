package s3

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
)

func init() {
	outputs.RegisterOutputPlugin("s3", New)
}

type s3Output struct {
	beatName string
	manager  fileManager
}

// New instantiates a new file output instance.
func New(beatName string, cfg *common.Config, _ int) (outputs.Outputer, error) {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	// disable bulk support in publisher pipeline
	cfg.SetInt("flush_interval", -1, -1)
	cfg.SetInt("bulk_max_size", -1, -1)

	output := &s3Output{beatName: beatName}
	if err := output.init(config); err != nil {
		return nil, err
	}
	return output, nil
}

func (out *s3Output) init(config config) error {
	out.manager.Path = config.Path
	out.manager.Name = config.Filename
	out.manager.Region = config.Region
	out.manager.Bucket = config.Bucket
	if out.manager.Name == "" {
		out.manager.Name = out.beatName
	}
	logp.Info("S3 output path set to: %v", out.manager.Path)
	logp.Info("S3 output base filename set to: %v", out.manager.Name)
	logp.Info("S3 output region set to: %v", out.manager.Region)
	logp.Info("S3 output bucket set to: %v", out.manager.Bucket)

	uploadeverybytes := uint64(config.UploadEveryKb) * 1024
	logp.Info("S3 upload every bytes set to: %v", uploadeverybytes)
	out.manager.UploadEveryBytes = &uploadeverybytes

	keepfiles := config.NumberOfFiles
	logp.Info("S3 number of files set to: %v", keepfiles)
	out.manager.KeepFiles = &keepfiles

	err := out.manager.createDirectory()
	if err != nil {
		return err
	}

	err = out.manager.checkIfConfigSane()
	if err != nil {
		return err
	}

	return nil
}

// Implement Outputer
func (out *s3Output) Close() error {
	return nil
}

func (out *s3Output) PublishEvent(
	sig op.Signaler,
	opts outputs.Options,
	data outputs.Data,
) error {
	jsonEvent, err := json.Marshal(data.Event)
	if err != nil {
		// mark as success so event is not sent again.
		op.SigCompleted(sig)

		logp.Err("S3 fail to json encode event(%v): %#v", err, data.Event)
		return err
	}

	err = out.manager.writeLine(jsonEvent)
	if err != nil {
		if opts.Guaranteed {
			logp.Critical("S3 unable to write events to file: %s", err)
		} else {
			logp.Err("S3 error when writing line to file: %s", err)
		}
	}
	op.Sig(sig, err)
	return err
}
