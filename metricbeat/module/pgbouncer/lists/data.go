package lists

import (
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	"github.com/elastic/beats/v7/libbeat/common/schema/mapstrstr"
)

var schema = s.Schema{
	"databases":     mapstrstr.Int("databases"),
	"users":         mapstrstr.Int("users"),
	"peers":         mapstrstr.Int("peers"),
	"pools":         mapstrstr.Int("pools"),
	"peer_pools":    mapstrstr.Int("peer_pools"),
	"free_clients":  mapstrstr.Int("free_clients"),
	"used_clients":  mapstrstr.Int("used_clients"),
	"login_clients": mapstrstr.Int("login_clients"),
	"free_servers":  mapstrstr.Int("free_servers"),
	"used_servers":  mapstrstr.Int("used_servers"),
	"dns_names":     mapstrstr.Int("dns_names"),
	"dns_zones":     mapstrstr.Int("dns_zones"),
}
