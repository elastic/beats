package dbstats

import (
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
)

var schema = s.Schema{
	"db":          c.Str("db"),
	"collections": c.Int("collections"),
	"objects":     c.Int("objects"),
	"avg_obj_size": s.Object{
		"bytes": c.Int("avgObjSize"),
	},
	"data_size": s.Object{
		"bytes": c.Int("dataSize"),
	},
	"storage_size": s.Object{
		"bytes": c.Int("storageSize"),
	},
	"num_extents": c.Int("numExtents"),
	"indexes":     c.Int("indexes"),
	"index_size": s.Object{
		"bytes": c.Int("indexSize"),
	},
	// mmapv1 only
	"ns_size_mb": s.Object{
		"mb": c.Int("nsSizeMB", s.Optional),
	},
	// mmapv1 only
	"file_size": s.Object{
		"bytes": c.Int("fileSize", s.Optional),
	},
	// mmapv1 only
	"data_file_version": c.Dict("dataFileVersion", s.Schema{
		"major": c.Int("major"),
		"minor": c.Int("minor"),
	}, c.DictOptional),
	// mmapv1 only
	"extent_free_list": c.Dict("extentFreeList", s.Schema{
		"num": c.Int("num"),
		"size": s.Object{
			"bytes": c.Int("size", s.Optional),
		},
	}, c.DictOptional),
}
