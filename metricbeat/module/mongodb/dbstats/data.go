package dbstats

import (
	s "github.com/elastic/beats/metricbeat/schema"
	c "github.com/elastic/beats/metricbeat/schema/mapstriface"
)

var schema = s.Schema{
	"db":           c.Str("db"),
	"collections":  c.Int("collections"),
	"objects":      c.Int("objects"),
	"avg_obj_size": c.Int("avgObjSize"),
	"data_size":    c.Int("dataSize"),
	"storage_size": c.Int("storageSize"),
	"num_extents":  c.Int("numExtents"),
	"indexes":      c.Int("indexes"),
	"index_size":   c.Int("indexSize"),
	// mmapv1 only
	"ns_size_mb": c.Int("nsSizeMB", s.Optional),
	// mmapv1 only
	"file_size": c.Int("fileSize", s.Optional),
	// mmapv1 only
	"data_file_version": c.Dict("dataFileVersion", s.Schema{
		"major": c.Int("major"),
		"minor": c.Int("minor"),
	}, c.DictOptional),
	// mmapv1 only
	"extent_free_list": c.Dict("extentFreeList", s.Schema{
		"num":  c.Int("num"),
		"size": c.Int("size"),
	}, c.DictOptional),
}

var eventMapping = schema.Apply
