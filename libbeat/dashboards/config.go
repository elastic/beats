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
	Snapshot       bool   `config:"snapshot"`
	SnapshotURL    string `config:"snapshot_url"`
}

var defaultConfig = Config{
	KibanaIndex: ".kibana",
}
var (
	defaultURLPattern  = "https://artifacts.elastic.co/downloads/beats/beats-dashboards/beats-dashboards-%s.zip"
	snapshotURLPattern = "https://snapshots.elastic.co/downloads/beats/beats-dashboards/beats-dashboards-%s-SNAPSHOT.zip"
)
