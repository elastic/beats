package keyspace

import (
	"strings"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstrstr"
	"github.com/elastic/beats/metricbeat/module/redis"
)

// Map data to MapStr
func eventsMapping(info map[string]string) []common.MapStr {
	events := []common.MapStr{}
	for key, space := range getKeyspaceStats(info) {
		space["id"] = key
		events = append(events, space)
	}

	return events
}

func getKeyspaceStats(info map[string]string) map[string]common.MapStr {
	keyspaceMap := findKeyspaceStats(info)
	return parseKeyspaceStats(keyspaceMap)
}

// findKeyspaceStats will grep for keyspace ("^db" keys) and return the resulting map
func findKeyspaceStats(info map[string]string) map[string]string {
	keyspace := map[string]string{}

	for k, v := range info {
		if strings.HasPrefix(k, "db") {
			keyspace[k] = v
		}
	}
	return keyspace
}

var schema = s.Schema{
	"keys":    c.Int("keys"),
	"expires": c.Int("expires"),
	"avg_ttl": c.Int("avg_ttl"),
}

// parseKeyspaceStats resolves the overloaded value string that Redis returns for keyspace
func parseKeyspaceStats(keyspaceMap map[string]string) map[string]common.MapStr {
	keyspace := map[string]common.MapStr{}
	for k, v := range keyspaceMap {

		// Extract out the overloaded values for db keyspace
		// fmt: info[db0] = keys=795341,expires=0,avg_ttl=0
		dbInfo := redis.ParseRedisLine(v, ",")

		if len(dbInfo) == 3 {
			db := map[string]interface{}{}
			for _, dbEntry := range dbInfo {
				stats := redis.ParseRedisLine(dbEntry, "=")

				if len(stats) == 2 {
					db[stats[0]] = stats[1]
				}
			}
			data, _ := schema.Apply(db)
			keyspace[k] = data
		}
	}
	return keyspace
}
