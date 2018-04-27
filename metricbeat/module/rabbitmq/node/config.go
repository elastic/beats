package node

const (
	configCollectNode    = "node"
	configCollectCluster = "cluster"
)

// Config for node metricset
type Config struct {
	// Collect mode
	// - `node` to collect metrics for endpoint only (default)
	// - `cluster` to collect metrics for all nodes in the cluster
	Collect string `config:"node.collect"`
}

var defaultConfig = Config{
	Collect: configCollectNode,
}
