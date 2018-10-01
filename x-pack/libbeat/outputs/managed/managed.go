// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package managed

import (
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/publisher"
)

// managed output is used as a placeholder for central management
// this output will cause the beat to pause the output until a real
// output is configured
// It's used both during startup (before we retrieve the current output settings)
// and can be also used to effectively pause the beat
type managed struct{}

func init() {
	outputs.RegisterType("managed", makeManaged)
}

func makeManaged(
	beat beat.Info,
	observer outputs.Observer,
	cfg *common.Config,
) (outputs.Group, error) {
	c := &managed{}
	return outputs.Success(0, 0, c)
}

func (c *managed) Close() error { return nil }
func (c *managed) Publish(batch publisher.Batch) error {
	time.Sleep(60 * time.Second)
	batch.Retry()
	return nil
}

func (c *managed) String() string {
	return "managed"
}
