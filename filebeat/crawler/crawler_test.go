// +build !integration

package crawler

import (
	"testing"

	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

func TestCrawlerStartError(t *testing.T) {
	crawler := Crawler{}
	channel := make(chan *input.FileEvent, 1)
	prospectorConfigs := []*common.Config{}

	error := crawler.Start(prospectorConfigs, channel)

	assert.Error(t, error)
}
