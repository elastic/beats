package outputs

import "github.com/elastic/beats/libbeat/common"

// ReadHostList reads a list of hosts to connect to from an configuration
// object. If the `workers` settings is > 1, each host is duplicated in the final
// host list by the number of `workers`.
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
