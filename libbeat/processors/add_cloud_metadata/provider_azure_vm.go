package add_cloud_metadata

import (
	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
)

// Azure VM Metadata Service
func newAzureVmMetadataFetcher(config *common.Config) (*metadataFetcher, error) {
	azMetadataURI := "/metadata/instance/compute?api-version=2017-04-02"
	azHeaders := map[string]string{"Metadata": "true"}
	azSchema := func(m map[string]interface{}) common.MapStr {
		out, _ := s.Schema{
			"instance_id":   c.Str("vmId"),
			"instance_name": c.Str("name"),
			"machine_type":  c.Str("vmSize"),
			"region":        c.Str("location"),
		}.Apply(m)
		return out
	}

	fetcher, err := newMetadataFetcher(config, "az", azHeaders, metadataHost, azSchema, azMetadataURI)
	return fetcher, err
}
