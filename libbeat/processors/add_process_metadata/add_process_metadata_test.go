package add_process_metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcessorString(t *testing.T) {
	p := &Processor{Config: defaultConfig}
	p.PIDFields = []string{"process.pid", "process.ppid"}
	assert.Equal(t, "add_process_metadata=[pid_fields=[process.pid,process.ppid], "+
		"metadata_types=[container_id], target.container_id=docker.container.id "+
		"target.cgroups=process.cgroups]",
		p.String())
}
