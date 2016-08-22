package modeutil

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs/mode"
	"github.com/elastic/beats/libbeat/outputs/mode/lb"
	"github.com/elastic/beats/libbeat/outputs/mode/single"
)

type ClientFactory func(host string) (mode.ProtocolClient, error)

type AsyncClientFactory func(string) (mode.AsyncProtocolClient, error)

type Settings struct {
	Failover     bool
	MaxAttempts  int
	WaitRetry    time.Duration
	Timeout      time.Duration
	MaxWaitRetry time.Duration
}

func NewConnectionMode(
	clients []mode.ProtocolClient,
	s Settings,
) (mode.ConnectionMode, error) {
	if s.Failover {
		clients = NewFailoverClient(clients)
	}

	maxSend := s.MaxAttempts
	wait := s.WaitRetry
	maxWait := s.MaxWaitRetry
	to := s.Timeout

	if len(clients) == 1 {
		return single.New(clients[0], maxSend, wait, to, maxWait)
	}
	return lb.NewSync(clients, maxSend, wait, to, maxWait)
}

func NewAsyncConnectionMode(
	clients []mode.AsyncProtocolClient,
	s Settings,
) (mode.ConnectionMode, error) {
	if s.Failover {
		clients = NewAsyncFailoverClient(clients)
	}
	return lb.NewAsync(clients, s.MaxAttempts, s.WaitRetry, s.Timeout, s.MaxWaitRetry)
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
		Hosts  []string `config:"hosts"  validate:"required"`
		Worker int      `config:"worker" validate:"min=1"`
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
