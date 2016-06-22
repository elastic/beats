// +build !integration

package crawler

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

func TestNewCrawlerNoProspectorsError(t *testing.T) {
	prospectorConfigs := []*common.Config{}

	_, error := New(nil, prospectorConfigs)

	assert.Error(t, error)
}
