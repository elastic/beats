package dbstats

import (
	s "github.com/elastic/beats/metricbeat/schema"
	c "github.com/elastic/beats/metricbeat/schema/mapstriface"
)

var schema = s.Schema{
	"db":              c.Str("db"),
	"collections":     c.Int("collections"),
	"objects":         c.Int("objects"),
	"avg_object_size": c.Int("avgObjectSize"),
	"data_size":       c.Int("dataSize"),
	"storage_size":    c.Int("storageSize"),
	"num_extents":     c.Int("numExtents"),
	"indexes":         c.Int("indexes"),
	"index_size":      c.Int("indexSize"),
	"file_size":       c.Int("fileSize"),
}

var eventMapping = schema.Apply
