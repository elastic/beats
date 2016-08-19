package main

import (
	"os"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/packetbeat/beater"

	// import support protocol modules
	_ "github.com/elastic/beats/packetbeat/protos/amqp"
	_ "github.com/elastic/beats/packetbeat/protos/cassandra"
	_ "github.com/elastic/beats/packetbeat/protos/dns"
	_ "github.com/elastic/beats/packetbeat/protos/http"
	_ "github.com/elastic/beats/packetbeat/protos/memcache"
	_ "github.com/elastic/beats/packetbeat/protos/mongodb"
	_ "github.com/elastic/beats/packetbeat/protos/mysql"
	_ "github.com/elastic/beats/packetbeat/protos/nfs"
	_ "github.com/elastic/beats/packetbeat/protos/pgsql"
	_ "github.com/elastic/beats/packetbeat/protos/redis"
	_ "github.com/elastic/beats/packetbeat/protos/thrift"
)

var Name = "packetbeat"

// Setups and Runs Packetbeat
func main() {
	if err := beat.Run(Name, "", beater.New); err != nil {
		os.Exit(1)
	}
}
