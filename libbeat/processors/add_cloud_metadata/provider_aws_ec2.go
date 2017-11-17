package add_cloud_metadata

import (
	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
)

// AWS EC2 Metadata Service
func newEc2MetadataFetcher(config *common.Config) (*metadataFetcher, error) {
	ec2InstanceIdentityURI := "/2014-02-25/dynamic/instance-identity/document"
	ec2Schema := func(m map[string]interface{}) common.MapStr {
		out, _ := s.Schema{
			"instance_id":       c.Str("instanceId"),
			"machine_type":      c.Str("instanceType"),
			"region":            c.Str("region"),
			"availability_zone": c.Str("availabilityZone"),
		}.Apply(m)
		return out
	}

	fetcher, err := newMetadataFetcher(config, "ec2", nil, metadataHost, ec2Schema, ec2InstanceIdentityURI)
	return fetcher, err
}
