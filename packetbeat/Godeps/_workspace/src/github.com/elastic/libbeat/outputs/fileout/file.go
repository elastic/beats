package fileout

import (
	"encoding/json"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/outputs"
)

func init() {
	outputs.RegisterOutputPlugin("file", FileOutputPlugin{})
}

type FileOutputPlugin struct{}

func (f FileOutputPlugin) NewOutput(
	beat string,
	config *outputs.MothershipConfig,
	topology_expire int,
) (outputs.Outputer, error) {
	output := &fileOutput{}
	err := output.init(beat, config, topology_expire)
	if err != nil {
		return nil, err
	}
	return output, nil
}

type fileOutput struct {
	rotator logp.FileRotator
}

func (out *fileOutput) init(beat string, config *outputs.MothershipConfig, topology_expire int) error {
	out.rotator.Path = config.Path
	out.rotator.Name = config.Filename
	if out.rotator.Name == "" {
		out.rotator.Name = beat
	}

	// disable bulk support
	configDisableInt := -1
	config.Flush_interval = &configDisableInt
	config.BulkMaxSize = &configDisableInt

	rotateeverybytes := uint64(config.Rotate_every_kb) * 1024
	if rotateeverybytes == 0 {
		rotateeverybytes = 10 * 1024 * 1024
	}
	out.rotator.RotateEveryBytes = &rotateeverybytes

	keepfiles := config.Number_of_files
	if keepfiles == 0 {
		keepfiles = 7
	}
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
	ts time.Time,
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
		logp.Err("Error when writing line to file: %s", err)
	}

	outputs.Signal(trans, err)
	return err
}
