package bundle

import (
	"net/http"
	"time"

	"github.com/elastic/beats/v7/libbeat/logp"
	csppolicies "github.com/elastic/csp-security-policies/bundle"
)

var address = "127.0.0.1:18080"

var ServerAddress = "http://" + address

var Config = `{
        "services": {
            "test": {
                "url": %q
            }
        },
        "bundles": {
            "test": {
                "resource": "/bundles/bundle.tar.gz"
            }
        },
        "decision_logs": {
            "console": true
        }
    }`

func StartServer() (*http.Server, error) {
	policies, err := csppolicies.CISKubernetes()
	if err != nil {
		return nil, err
	}

	bundleServer := csppolicies.NewServer()
	err = bundleServer.HostBundle("bundle.tar.gz", policies)
	if err != nil {
		return nil, err
	}

	srv := &http.Server{
		Addr:         address,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      bundleServer,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			logp.L().Errorf("bundle server closed: %v", err)
		}
	}()

	return srv, nil
}
