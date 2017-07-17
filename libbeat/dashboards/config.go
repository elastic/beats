package dashboards

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
}

var defaultConfig = Config{
	KibanaIndex: ".kibana",
}
var (
	defaultDirectory = "kibana"
)
