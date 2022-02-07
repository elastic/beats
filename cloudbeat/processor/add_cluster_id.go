package add_cluster_id

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/processors"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
)

func init() {
	processors.RegisterPlugin("add_cluster_id", New)
	jsprocessor.RegisterPlugin("AddClusterID", New)
}

const processorName = "add_cluster_id"

type addClusterID struct {
	config        config
	clusterHelper *ClusterHelper
}

// New constructs a new Add ID processor.
func New(cfg *common.Config) (processors.Processor, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, makeErrConfigUnpack(err)
	}

	clusterHelper, err := newClusterHelper()
	if err != nil {
		return nil, err
	}
	p := &addClusterID{
		config,
		clusterHelper,
	}

	return p, nil
}

// Run enriches the given event with an ID
func (p *addClusterID) Run(event *beat.Event) (*beat.Event, error) {
	clusterId := p.clusterHelper.ClusterId()

	if _, err := event.PutValue(p.config.TargetField, clusterId); err != nil {
		return nil, makeErrComputeID(err)
	}

	return event, nil
}

func (p *addClusterID) String() string {
	return fmt.Sprintf("%v=[target_field=[%v]]", processorName, p.config.TargetField)
}
