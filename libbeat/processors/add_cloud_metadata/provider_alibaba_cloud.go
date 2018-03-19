package add_cloud_metadata

import "github.com/elastic/beats/libbeat/common"

// Alibaba Cloud Metadata Service
// Document https://help.aliyun.com/knowledge_detail/49122.html
func newAlibabaCloudMetadataFetcher(c *common.Config) (*metadataFetcher, error) {
	ecsMetadataHost := "100.100.100.200"
	ecsMetadataInstanceIDURI := "/latest/meta-data/instance-id"
	ecsMetadataRegionURI := "/latest/meta-data/region-id"
	ecsMetadataZoneURI := "/latest/meta-data/zone-id"

	ecsSchema := func(m map[string]interface{}) common.MapStr {
		return common.MapStr(m)
	}

	urls, err := getMetadataURLs(c, ecsMetadataHost, []string{
		ecsMetadataInstanceIDURI,
		ecsMetadataRegionURI,
		ecsMetadataZoneURI,
	})
	if err != nil {
		return nil, err
	}
	responseHandlers := map[string]responseHandler{
		urls[0]: func(all []byte, result *result) error {
			result.metadata["instance_id"] = string(all)
			return nil
		},
		urls[1]: func(all []byte, result *result) error {
			result.metadata["region"] = string(all)
			return nil
		},
		urls[2]: func(all []byte, result *result) error {
			result.metadata["availability_zone"] = string(all)
			return nil
		},
	}
	fetcher := &metadataFetcher{"ecs", nil, responseHandlers, ecsSchema}
	return fetcher, nil
}
