package outputs

// Fail helper can be used by output factories, to create a failure response when
// loading an output must return an error.
func Fail(err error) (Group, error) { return Group{}, err }

// Success create a valid output Group response for a set of client instances.
func Success(batchSize, retry int, clients ...Client) (Group, error) {
	return Group{
		Clients:   clients,
		BatchSize: batchSize,
		Retry:     retry,
	}, nil
}

// NetworkClients converts a list of NetworkClient instances into []Client.
func NetworkClients(netclients []NetworkClient) []Client {
	clients := make([]Client, len(netclients))
	for i, n := range netclients {
		clients[i] = n
	}
	return clients
}

func SuccessNet(loadbalance bool, batchSize, retry int, netclients []NetworkClient) (Group, error) {
	if !loadbalance {
		return Success(batchSize, retry, NewFailoverClient(netclients))
	}

	clients := NetworkClients(netclients)
	return Success(batchSize, retry, clients...)
}
