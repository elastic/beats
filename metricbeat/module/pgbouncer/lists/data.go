package lists

import (
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstrstr"
)

var schema = s.Schema{
	"databases":     c.Int("databases"),
	"users":         c.Int("users"),
	"peers":         c.Int("peers"),
	"pools":         c.Int("pools"),
	"peer_pools":    c.Int("peer_pools"),
	"free_clients":  c.Int("free_clients"),
	"used_clients":  c.Int("used_clients"),
	"login_clients": c.Int("login_clients"),
	"free_servers":  c.Int("free_servers"),
	"used_servers":  c.Int("used_servers"),
	"dns_names":     c.Int("dns_names"),
	"dns_zones":     c.Int("dns_zones"),
}
