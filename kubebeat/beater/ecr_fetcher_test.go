package beater

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEksDataFetcherFetchECR(t *testing.T) {

	//Creating a new evaluation parser
	eksFetcher:= ECRDataFetcher{}

	results, err := eksFetcher.DescribeAllRepositories()
	if err != nil {
		assert.Fail(t, "error during parsing of the json", err)
	}

	for _, event := range results {

		assert.NotNil(t, event, `event timestamp is not correct`)
	}
}
