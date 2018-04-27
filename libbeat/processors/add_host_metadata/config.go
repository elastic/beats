package add_host_metadata

//import (
//	"github.com/elastic/beats/libbeat/processors"
//)

// Config for add_host_metadata processor.
type Config struct {
	NetInfoEnabled bool `config:"netinfo.enabled"` // Add IP and MAC to event
}

func defaultConfig() Config {
	return Config{
		//		netInfoEnabled: "false",
	}
}
