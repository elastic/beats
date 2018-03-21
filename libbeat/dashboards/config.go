package dashboards

import "time"

type Config struct {
	Enabled        bool   `config:"enabled"`
	KibanaIndex    string `config:"kibana_index"`
	Index          string `config:"index"`
	Dir            string `config:"directory"`
	File           string `config:"file"`
	Beat           string `config:"beat"`
	URL            string `config:"url"`
	OnlyDashboards bool   `config:"only_dashboards"`
	OnlyIndex      bool   `config:"only_index"`
	AlwaysKibana   bool   `config:"always_kibana"`
	Retry          *Retry `config:"retry"`
}

type Retry struct {
	Enabled  bool          `config:"enabled"`
	Interval time.Duration `config:"interval"`
	Maximum  uint          `config:"maximum"`
}

var defaultConfig = Config{
	KibanaIndex:  ".kibana",
	AlwaysKibana: false,
	Retry: &Retry{
		Enabled:  false,
		Interval: time.Second,
		Maximum:  0,
	},
}
var (
	defaultDirectory = "kibana"
)
