package add_cloud_metadata

import "github.com/elastic/beats/libbeat/common"

// Tencent Cloud Metadata Service
// Document https://www.qcloud.com/document/product/213/4934
func newQcloudMetadataFetcher(c *common.Config) (*metadataFetcher, error) {
	qcloudMetadataHost := "metadata.tencentyun.com"
	qcloudMetadataInstanceIDURI := "/meta-data/instance-id"
	qcloudMetadataRegionURI := "/meta-data/placement/region"
	qcloudMetadataZoneURI := "/meta-data/placement/zone"

	qcloudSchema := func(m map[string]interface{}) common.MapStr {
		return common.MapStr(m)
	}

	urls, err := getMetadataURLs(c, qcloudMetadataHost, []string{
		qcloudMetadataInstanceIDURI,
		qcloudMetadataRegionURI,
		qcloudMetadataZoneURI,
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
	fetcher := &metadataFetcher{"qcloud", nil, responseHandlers, qcloudSchema}
	return fetcher, nil
}
