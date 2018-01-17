package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
)

// Start starts the metrics api endpoint on the configured host and port
func Start(cfg *common.Config, info beat.Info) {
	cfgwarn.Experimental("Metrics endpoint is enabled.")
	config := DefaultConfig
	cfg.Unpack(&config)

	logp.Info("Starting stats endpoint")
	go func() {
		mux := http.NewServeMux()

		// register handlers
		mux.HandleFunc("/", rootHandler(info))
		mux.HandleFunc("/stats", statsHandler)

		url := config.Host + ":" + strconv.Itoa(config.Port)
		logp.Info("Metrics endpoint listening on: %s", url)
		endpoint := http.ListenAndServe(url, mux)
		logp.Info("finished starting stats endpoint: %v", endpoint)
	}()
}

func rootHandler(info beat.Info) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Return error page
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		data := common.MapStr{
			"version":  info.Version,
			"beat":     info.Beat,
			"name":     info.Name,
			"uuid":     info.UUID,
			"hostname": info.Hostname,
		}

		print(w, data, r.URL)
	}
}

// statsHandler report expvar and all libbeat/monitoring metrics
func statsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	data := monitoring.CollectStructSnapshot(nil, monitoring.Full, false)

	print(w, data, r.URL)
}

func print(w http.ResponseWriter, data common.MapStr, u *url.URL) {
	query := u.Query()
	if _, ok := query["pretty"]; ok {
		fmt.Fprintf(w, data.StringToPrint())
	} else {
		fmt.Fprintf(w, data.String())
	}
}
