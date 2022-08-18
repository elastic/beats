package tds

import (
	"github.com/elastic/beats/v7/libbeat/asset"
)

func init() {
	if err := asset.SetFields("packetbeat", "tds", asset.ModuleFieldsPri, AssetTds); err != nil {
		panic(err)
	}
}

// AssetTds returns asset data.
// This is the base64 encoded zlib format compressed contents of protos/tds.
func AssetTds() string {
	return ""
}
