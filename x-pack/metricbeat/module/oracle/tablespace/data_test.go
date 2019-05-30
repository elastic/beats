package tablespace

import (
	"fmt"
	"github.com/kr/pretty"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEventMapping(t *testing.T) {
	t.Run("Happy Path", func(t *testing.T) {
		m := MetricSet{extractor: &happyMockExtractor{}}

		events, err := m.eventMapping()
		assert.NoError(t, err)

		fmt.Printf("Total %d events\n", len(events))
		for _, event := range events {
			pretty.Println(event.MetricSetFields)
		}
	})

	t.Run("Error Paths", func(t *testing.T) {
		t.Run("data files", func(t *testing.T) {
			m := MetricSet{extractor: &errorDataFilesMockExtractor{}}

			_, err := m.eventMapping()
			assert.Error(t, err)
		})

		t.Run("free space data", func(t *testing.T) {
			m := MetricSet{extractor: &errorFreeSpaceDataMockExtractor{}}

			_, err := m.eventMapping()
			assert.Error(t, err)
		})

		t.Run("temp free space data", func(t *testing.T) {
			m := MetricSet{extractor: &errorTempFreeSpaceDataMockExtractor{}}

			_, err := m.eventMapping()
			assert.Error(t, err)
		})
	})
}
