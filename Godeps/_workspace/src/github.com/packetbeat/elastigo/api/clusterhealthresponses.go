package api

type ClusterHealthResponse struct {
	ClusterName         string `json:"cluster_name"`
	Status              string `json:"status"`
	TimedOut            bool   `json:"timed_out"`
	NumberOfNodes       int    `json:"number_of_nodes"`
	NumberOfDataNodes   int    `json:"number_of_data_nodes"`
	ActivePrimaryShards int    `json:"active_primary_shards"`
	ActiveShards        int    `json:"active_shards"`
	RelocatingShards    int    `json:"relocating_shards"`
	InitializingShards  int    `json:"initializing_shards"`
	UnassignedShards    int    `json:"unassigned_shards"`
}

type ClusterStateResponse struct {
	ClusterName string                             `json:"cluster_name"`
	MasterNode  string                             `json:"master_node"`
	Nodes       map[string]ClusterStateNodeReponse `json:"nodes"`
	// TODO: Metadata
	// TODO: Routing Table
	// TODO: Routing Nodes
	// TODO: Allocations

}

type ClusterStateNodeReponse struct {
	Name             string `json:"name"`
	TransportAddress string `json:"transport_address"`
	// TODO: Attributes
}

type ClusterStateMetadataResponse struct {
	// TODO: templates
	// TODO: indices
}

type ClusterStateRoutingTableResponse struct {
	// TODO: unassigned
	//
}
