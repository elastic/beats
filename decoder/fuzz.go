// +build gofuzz

package decoder

import (
	"os"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/publisher"

	"github.com/elastic/packetbeat/config"
	"github.com/elastic/packetbeat/protos"
	"github.com/elastic/packetbeat/protos/dns"
	"github.com/elastic/packetbeat/protos/http"
	// "github.com/elastic/packetbeat/protos/icmp"
	"github.com/elastic/packetbeat/protos/memcache"
	"github.com/elastic/packetbeat/protos/mongodb"
	"github.com/elastic/packetbeat/protos/mysql"
	"github.com/elastic/packetbeat/protos/pgsql"
	"github.com/elastic/packetbeat/protos/redis"
	"github.com/elastic/packetbeat/protos/tcp"
	// "github.com/elastic/packetbeat/protos/thrift"
	"github.com/elastic/packetbeat/protos/udp"

	"github.com/tsg/gopacket"
	"github.com/tsg/gopacket/layers"
)

// possible return values
//  1 if the input is interesting in some way (for example, it was parsed successfully, that is, it is lexically correct, go-fuzz will give more priority to such inputs)
// -1 if the input must not be added to corpus even if gives new coverage
//  0 otherwise
func Fuzz(data []byte) int {
	packet := gopacket.NewPacket(data, layers.LinkTypeEthernet, gopacket.Default)
	if packet.ErrorLayer() != nil {
		return 0
	}

	decoder := newTestDecoder()
	decoder.DecodePacketData(packet.Data(), &packet.Metadata().CaptureInfo)

	return 1
}

func newTestDecoder() *DecoderStruct {
	publisher := new(NoOpPublisher)

	for proto, plugin := range EnabledProtocolPlugins {
		err := plugin.Init(false, publisher)
		if err != nil {
			logp.Critical("Initializing plugin %s failed: %v", proto, err)
			os.Exit(1)
		}
		protos.Protos.Register(proto, plugin)
	}

	var err error

	// testMode := false
	// icmpProc, err := icmp.NewIcmp(testMode, publisher)
	// if err != nil {
	// 	logp.Critical(err.Error())
	// 	os.Exit(1)
	// }

	tcpProc, err := tcp.NewTcp(&protos.Protos)
	if err != nil {
		logp.Critical(err.Error())
		os.Exit(1)
	}

	udpProc, err := udp.NewUdp(&protos.Protos)
	if err != nil {
		logp.Critical(err.Error())
		os.Exit(1)
	}

	// decoder, err := NewDecoder(layers.LinkTypeEthernet, icmpProc, icmpProc, tcpProc, udpProc)
	decoder, err := NewDecoder(layers.LinkTypeEthernet, tcpProc, udpProc)
	if err != nil {
		logp.Critical(err.Error())
		os.Exit(1)
	}

	return decoder
}

type NoOpPublisher struct{}

func (c NoOpPublisher) PublishEvent(event common.MapStr, opts ...publisher.ClientOption) bool {
	return true
}

func (c NoOpPublisher) PublishEvents(events []common.MapStr, opts ...publisher.ClientOption) bool {
	return true
}

var EnabledProtocolPlugins map[protos.Protocol]protos.ProtocolPlugin = map[protos.Protocol]protos.ProtocolPlugin{
	protos.HttpProtocol:     new(http.Http),
	protos.MemcacheProtocol: new(memcache.Memcache),
	protos.MysqlProtocol:    new(mysql.Mysql),
	protos.PgsqlProtocol:    new(pgsql.Pgsql),
	protos.RedisProtocol:    new(redis.Redis),
	// protos.ThriftProtocol:   new(thrift.Thrift),
	protos.MongodbProtocol:  new(mongodb.Mongodb),
	protos.DnsProtocol:      new(dns.Dns),
}

func init() {
	pTrue := new(bool)
	*pTrue = true

	config.ConfigSingleton = config.Config{
		Protocols: config.Protocols{
			Dns: config.Dns{
				ProtocolCommon: config.ProtocolCommon{
					Ports:        []int{53},
					SendRequest:  pTrue,
					SendResponse: pTrue,
				},
				Include_authorities: pTrue,
				Include_additionals: pTrue,
			},
			Http: config.Http{
				ProtocolCommon: config.ProtocolCommon{
					Ports:        []int{80, 8080, 8000, 5000, 8002},
					SendRequest:  pTrue,
					SendResponse: pTrue,
				},
				Send_all_headers:     pTrue,
				Split_cookie:         pTrue,
				Redact_authorization: pTrue,
				// Send_headers       []string
				// Real_ip_header     *string
				// Include_body_for   []string
				// Hide_keywords      []string
			},
			Memcache: config.Memcache{
				ProtocolCommon: config.ProtocolCommon{
					Ports:        []int{11211},
					SendRequest:  pTrue,
					SendResponse: pTrue,
				},
				ParseUnknown: true,
				// MaxValues             int
				// MaxBytesPerValue      int
				// UdpTransactionTimeout uint
				// TcpTransactionTimeout uint
			},
			Mysql: config.Mysql{
				ProtocolCommon: config.ProtocolCommon{
					Ports:        []int{3306},
					SendRequest:  pTrue,
					SendResponse: pTrue,
				},
				// Max_row_length *int
				// Max_rows       *int
			},
			Mongodb: config.Mongodb{
				ProtocolCommon: config.ProtocolCommon{
					Ports:        []int{27017},
					SendRequest:  pTrue,
					SendResponse: pTrue,
				},
				// Max_doc_length *int
				// Max_docs       *int
			},
			Pgsql: config.Pgsql{
				ProtocolCommon: config.ProtocolCommon{
					Ports:        []int{5432},
					SendRequest:  pTrue,
					SendResponse: pTrue,
				},
				// Max_row_length *int
				// Max_rows       *int
			},
			Redis: config.Redis{
				ProtocolCommon: config.ProtocolCommon{
					Ports:        []int{6379},
					SendRequest:  pTrue,
					SendResponse: pTrue,
				},
			},
			Thrift: config.Thrift{
				ProtocolCommon: config.ProtocolCommon{
					Ports:        []int{9090},
					SendRequest:  pTrue,
					SendResponse: pTrue,
				},
				Capture_reply:     pTrue,
				Obfuscate_strings: pTrue,
				// String_max_size            *int
				// Collection_max_size        *int
				// Drop_after_n_struct_fields *int
				// Transport_type             *string
				// Protocol_type              *string
				// Idl_files                  []string
			},
		},
	}
}
