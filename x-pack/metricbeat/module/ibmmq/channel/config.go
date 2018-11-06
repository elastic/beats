// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package channel

type ConnectionConfig struct {
	ClientMode bool
	UserId     string
	Password   string
}

type mqConfig struct {
	QueueManager       string `config:"queueManager"`
	RemoteQueueManager string `config:"remoteQueueManager"`
	Queue              string `config:"queue"`
	Channel            string `config:"channel"`
	QMgrStat           bool   `config:"queueManagerStatus"`
	PubSub             bool   `config:"pubSub"`
	Custom             string `config:"custome"`
	CC                 ConnectionConfig
}

var (
	DefaultConfig = mqConfig{
		PubSub:             false,
		QMgrStat:           true,
		RemoteQueueManager: "",
		Queue:              "*",
		Channel:            "*",
		Custom:             "",
	}
)
