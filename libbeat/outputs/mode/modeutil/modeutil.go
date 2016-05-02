package modeutil

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs/mode"
)

type ClientFactory func(host string) (mode.ProtocolClient, error)

type AsyncClientFactory func(string) (mode.AsyncProtocolClient, error)

func NewConnectionMode(
	clients []mode.ProtocolClient,
	failover bool,
	maxAttempts int,
	waitRetry, timeout, maxWaitRetry time.Duration,
) (mode.ConnectionMode, error) {
	if failover {
		clients = NewFailoverClient(clients)
	}

	if len(clients) == 1 {
		return mode.NewSingleConnectionMode(clients[0], maxAttempts, waitRetry, timeout, maxWaitRetry)
	}
	return mode.NewLoadBalancerMode(clients, maxAttempts,
		waitRetry, timeout, maxWaitRetry)
}

func NewAsyncConnectionMode(
	clients []mode.AsyncProtocolClient,
	failover bool,
	maxAttempts int,
	waitRetry, timeout, maxWaitRetry time.Duration,
) (mode.ConnectionMode, error) {
	if failover {
		clients = NewAsyncFailoverClient(clients)
	}
	return mode.NewAsyncLoadBalancerMode(
		clients, maxAttempts, waitRetry, timeout, maxWaitRetry)
}

// MakeClients will create a list from of ProtocolClient instances from
// outputer configuration host list and client factory function.
func MakeClients(
	config *common.Config,
	newClient ClientFactory,
) ([]mode.ProtocolClient, error) {
	hosts, err := ReadHostList(config)
	if err != nil {
		return nil, err
	}
	if len(hosts) == 0 {
		return nil, mode.ErrNoHostsConfigured
	}

	clients := make([]mode.ProtocolClient, 0, len(hosts))
	for _, host := range hosts {
		client, err := newClient(host)
		if err != nil {
			// on error destroy all client instance created
			for _, client := range clients {
				_ = client.Close() // ignore error
			}
			return nil, err
		}
		clients = append(clients, client)
	}
	return clients, nil
}

func MakeAsyncClients(
	config *common.Config,
	newClient AsyncClientFactory,
) ([]mode.AsyncProtocolClient, error) {
	hosts, err := ReadHostList(config)
	if err != nil {
		return nil, err
	}
	if len(hosts) == 0 {
		return nil, mode.ErrNoHostsConfigured
	}

	clients := make([]mode.AsyncProtocolClient, 0, len(hosts))
	for _, host := range hosts {
		client, err := newClient(host)
		if err != nil {
			// on error destroy all client instance created
			for _, client := range clients {
				_ = client.Close() // ignore error
			}
			return nil, err
		}
		clients = append(clients, client)
	}
	return clients, nil
}

func ReadHostList(cfg *common.Config) ([]string, error) {
	config := struct {
		Hosts  []string `config:"hosts"`
		Worker int      `config:"worker"`
	}{
		Worker: 1,
	}

	err := cfg.Unpack(&config)
	if err != nil {
		return nil, err
	}

	lst := config.Hosts
	if len(lst) == 0 || config.Worker <= 1 {
		return lst, nil
	}

	// duplicate entries config.Workers times
	hosts := make([]string, 0, len(lst)*config.Worker)
	for _, entry := range lst {
		for i := 0; i < config.Worker; i++ {
			hosts = append(hosts, entry)
		}
	}

	return hosts, nil
}
