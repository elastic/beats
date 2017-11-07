package add_cloud_metadata

import (
	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
)

// Google GCE Metadata Service
func newGceMetadataFetcher(config *common.Config) (*metadataFetcher, error) {
	gceMetadataURI := "/computeMetadata/v1/?recursive=true&alt=json"
	gceHeaders := map[string]string{"Metadata-Flavor": "Google"}
	gceSchema := func(m map[string]interface{}) common.MapStr {
		out := common.MapStr{}

		if instance, ok := m["instance"].(map[string]interface{}); ok {
			s.Schema{
				"instance_id":       c.StrFromNum("id"),
				"instance_name":     c.Str("name"),
				"machine_type":      c.Str("machineType"),
				"availability_zone": c.Str("zone"),
			}.ApplyTo(out, instance)
		}

		if project, ok := m["project"].(map[string]interface{}); ok {
			s.Schema{
				"project_id": c.Str("projectId"),
			}.ApplyTo(out, project)
		}

		return out
	}

	fetcher, err := newMetadataFetcher(config, "gce", gceHeaders, metadataHost, gceSchema, gceMetadataURI)
	return fetcher, err
}
