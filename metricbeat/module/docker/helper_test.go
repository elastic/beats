package docker

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"

	"github.com/stretchr/testify/assert"
)

func TestDeDotLabels(t *testing.T) {
	labels := map[string]string{
		"com.docker.swarm.task":      "",
		"com.docker.swarm.task.id":   "1",
		"com.docker.swarm.task.name": "foobar",
	}

	t.Run("dedot enabled", func(t *testing.T) {
		result := DeDotLabels(labels, true)
		assert.Equal(t, common.MapStr{
			"com_docker_swarm_task":      "",
			"com_docker_swarm_task_id":   "1",
			"com_docker_swarm_task_name": "foobar",
		}, result)
	})

	t.Run("dedot disabled", func(t *testing.T) {
		result := DeDotLabels(labels, false)
		assert.Equal(t, common.MapStr{
			"com": common.MapStr{
				"docker": common.MapStr{
					"swarm": common.MapStr{
						"task": common.MapStr{
							"value": "",
							"id":    "1",
							"name":  "foobar",
						},
					},
				},
			},
		}, result)
	})
}
