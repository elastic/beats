package kubernetes

type KubeClientOptions struct {
	QPS   float32 `config:"qps"`
	Burst int     `config:"burst"`
}
