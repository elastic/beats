package mem

import (
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstrstr"
)

var schema = s.Schema{
	"user_cache": s.Object{
		"size":     c.Int("size"),
		"used":     c.Int("used"),
		"free":     c.Int("free"),
		"memtotal": c.Int("memtotal"),
	},
	"credentials_cache": s.Object{
		"size":     c.Int("size"),
		"used":     c.Int("used"),
		"free":     c.Int("free"),
		"memtotal": c.Int("memtotal"),
	},
	"db_cache": s.Object{
		"size":     c.Int("size"),
		"used":     c.Int("used"),
		"free":     c.Int("free"),
		"memtotal": c.Int("memtotal"),
	},
	"peer_cache": s.Object{
		"size":     c.Int("size"),
		"used":     c.Int("used"),
		"free":     c.Int("free"),
		"memtotal": c.Int("memtotal"),
	},
	"peer_pool_cache": s.Object{
		"size":     c.Int("size"),
		"used":     c.Int("used"),
		"free":     c.Int("free"),
		"memtotal": c.Int("memtotal"),
	},
	"pool_cache": s.Object{
		"size":     c.Int("size"),
		"used":     c.Int("used"),
		"free":     c.Int("free"),
		"memtotal": c.Int("memtotal"),
	},
	"outstanding_request_cache": s.Object{
		"size":     c.Int("size"),
		"used":     c.Int("used"),
		"free":     c.Int("free"),
		"memtotal": c.Int("memtotal"),
	},
	"server_cache": s.Object{
		"size":     c.Int("size"),
		"used":     c.Int("used"),
		"free":     c.Int("free"),
		"memtotal": c.Int("memtotal"),
	},
	"iobuf_cache": s.Object{
		"size":     c.Int("size"),
		"used":     c.Int("used"),
		"free":     c.Int("free"),
		"memtotal": c.Int("memtotal"),
	},
	"var_list_cache": s.Object{
		"size":     c.Int("size"),
		"used":     c.Int("used"),
		"free":     c.Int("free"),
		"memtotal": c.Int("memtotal"),
	},
	"server_prepared_statement_cache": s.Object{
		"size":     c.Int("size"),
		"used":     c.Int("used"),
		"free":     c.Int("free"),
		"memtotal": c.Int("memtotal"),
	},
}
