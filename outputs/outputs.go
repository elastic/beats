package outputs

import "time"

type MapStr map[string]interface{}

type OutputInterface interface {
	PublishIPs(name string, localAddrs []string) error
	GetNameByIP(ip string) string
	PublishEvent(event *Event) error
}

type Event struct {
	Timestamp    time.Time `json:"timestamp"`
	Type         string    `json:"type"`
	Method       string    `json:"method"`
	Query        string    `json:"query"`
	Path         string    `json:"path"`
	Agent        string    `json:"agent"`
	Src_ip       string    `json:"client_ip"`
	Src_port     uint16    `json:"client_port"`
	Src_proc     string    `json:"client_proc"`
	Real_ip      string    `json:"real_ip"`
	Src_country  string    `json:"country"`
	Src_server   string    `json:"client_server"`
	Dst_ip       string    `json:"ip"`
	Dst_port     uint16    `json:"port"`
	Dst_proc     string    `json:"proc"`
	Dst_server   string    `json:"server"`
	ResponseTime int32     `json:"responsetime"`
	Status       string    `json:"status"`
	RequestRaw   string    `json:"request_raw"`
	ResponseRaw  string    `json:"response_raw"`
	Tags         string    `json:"tags"`
	BytesOut     uint64    `json:"bytes_out"`
	BytesIn      uint64    `json:"bytes_in"`

	Mysql  MapStr `json:"mysql"`
	Http   MapStr `json:"http"`
	Redis  MapStr `json:"redis"`
	Pgsql  MapStr `json:"pgsql"`
	Thrift MapStr `json:"thrift"`
}

type MothershipConfig struct {
	Enabled            bool
	Save_topology      bool
	Host               string
	Port               int
	Protocol           string
	Username           string
	Password           string
	Index              string
	Path               string
	Db                 int
	Db_topology        int
	Timeout            int
	Reconnect_interval int
	Filename           string
	Rotate_every_kb    int
	Number_of_files    int
	DataType           string
	Flush_interval     int
}
