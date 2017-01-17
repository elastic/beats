package nifi

import (
	"fmt"
	"net/http"

	"github.com/elastic/beats/metricbeat/mb"
)

func init() {
	// Register the ModuleFactory function for the "mongodb" module.
	if err := mb.Registry.AddModule("nifi", NewModule); err != nil {
		panic(err)
	}
}

// NewModule creates a new mb.Module instance and validates that at least one host has been
// specified
func NewModule(base mb.BaseModule) (mb.Module, error) {
	// Validate that at least one host has been specified.
	config := struct {
		Hosts []string `config:"hosts"    validate:"nonzero,required"`
	}{}
	if err := base.UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &base, nil
}

// IsCluster ...
func IsCluster(host string, client *http.Client) bool {
	url := fmt.Sprintf("http://%s/nifi-api/controller/cluster", host)

	req, _ := http.NewRequest("GET", url, nil)

	resp, _ := client.Do(req)

	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return true
	}
	return false
}

// GetNodeMap ...
func GetNodeMap(host string, client *http.Client) map[string]string {
	url := fmt.Sprintf("https://%s/nifi-api/controller/cluster", host)

	req, _ := http.NewRequest("GET", url, nil)

	resp, _ := client.Do(req)

	defer resp.Body.Close()

}
