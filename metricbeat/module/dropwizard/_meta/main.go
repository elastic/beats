package main

import (
	"fmt"
	"log"
	"net/http"
)

var response = `{"version":"4.0.0","gauges":{"my_gauge{this=that}":{"value":565}},"counters":{"my_counter":{"count":565},"my_counter2{this=that}":{"count":565}},"histograms":{"my_hist":{"count":565,"max":564,"mean":563.3148706761577,"min":0,"p50":564.0,"p75":564.0,"p95":564.0,"p98":564.0,"p99":564.0,"p999":564.0,"stddev":1.0747916190718627}},"meters":{"my_meter":{"count":0,"m1_rate":0.0,"m5_rate":0.0,"m15_rate":0.0,"mean_rate":0.0,"units":"events/second"}},"timers":{"my_timer{this=that}":{"count":0,"max":0.0,"mean":0.0,"min":0.0,"p50":0.0,"p75":0.0,"p95":0.0,"p98":0.0,"p99":0.0,"p999":0.0,"stddev":0.0,"m1_rate":0.0,"m5_rate":0.0,"m15_rate":0.0,"mean_rate":0.0,"duration_units":"seconds","rate_units":"calls/second"}}}`

func sendResponse(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, response)
}

func main() {

	http.HandleFunc("/metrics/metrics", sendResponse) // set router
	err := http.ListenAndServe(":9090", nil)          // set listen port
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
