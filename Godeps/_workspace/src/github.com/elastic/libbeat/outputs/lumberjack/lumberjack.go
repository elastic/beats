package lumberjack

import (
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/outputs"
)

func init() {
	outputs.RegisterOutputPlugin("lumberjack", lumberjackOutputPlugin{})
}

type lumberjackOutputPlugin struct{}

func (p lumberjackOutputPlugin) NewOutput(
	beat string,
	config outputs.MothershipConfig,
	topology_expire int,
) (outputs.Outputer, error) {
	output := &lumberjack{}
	err := output.init(beat, config, topology_expire)
	if err != nil {
		return nil, err
	}
	return output, nil
}

type lumberjack struct{}

func (lj *lumberjack) init(
	beat string,
	config outputs.MothershipConfig,
	topology_expire int,
) error {
	return nil
}

func (out *lumberjack) PublishEvent(ts time.Time, event common.MapStr) error {
	return nil
}
