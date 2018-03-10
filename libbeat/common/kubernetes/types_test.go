package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPodContainerStatus_GetContainerID(t *testing.T) {
	tests := []struct {
		status *PodContainerStatus
		result string
	}{
		// Check to see if x://y is parsed to return y as the container id
		{
			status: &PodContainerStatus{
				Name:        "foobar",
				ContainerID: "docker://abc",
				Image:       "foobar:latest",
			},
			result: "abc",
		},
		// Check to see if x://y is not the format then "" is returned
		{
			status: &PodContainerStatus{
				Name:        "foobar",
				ContainerID: "abc",
				Image:       "foobar:latest",
			},
			result: "",
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.status.GetContainerID(), test.result)
	}
}

func TestPodContainerStatus_GetContainerIDWithRuntime(t *testing.T) {
	tests := []struct {
		status *PodContainerStatus
		result string
	}{
		// Check to see if x://y is parsed to return x as the runtime
		{
			status: &PodContainerStatus{
				Name:        "foobar",
				ContainerID: "docker://abc",
				Image:       "foobar:latest",
			},
			result: "docker",
		},
		// Check to see if x://y is not the format then "" is returned
		{
			status: &PodContainerStatus{
				Name:        "foobar",
				ContainerID: "abc",
				Image:       "foobar:latest",
			},
			result: "",
		},
	}

	for _, test := range tests {
		_, runtime := test.status.GetContainerIDWithRuntime()
		assert.Equal(t, runtime, test.result)
	}
}
