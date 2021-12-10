// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"os"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control/client"
)

var testStatus = &client.AgentStatus{
	Status:  client.Healthy,
	Message: "",
	Applications: []*client.ApplicationStatus{{
		ID:      "id_1",
		Name:    "filebeat",
		Status:  client.Healthy,
		Message: "Running",
		Payload: nil,
	}, {
		ID:      "id_2",
		Name:    "metricbeat",
		Status:  client.Healthy,
		Message: "Running",
		Payload: nil,
	}, {
		ID:      "id_3",
		Name:    "filebeat_monitoring",
		Status:  client.Healthy,
		Message: "Running",
		Payload: nil,
	}, {
		ID:      "id_4",
		Name:    "metricbeat_monitoring",
		Status:  client.Healthy,
		Message: "Running",
		Payload: nil,
	},
	},
}

func ExamplehumanStatusOutput() {
	humanStatusOutput(os.Stdout, testStatus)
	// Output:
	// Status: HEALTHY
	// Message: (no message)
	// Applications:
	//   * filebeat               (HEALTHY)
	//                            Running
	//   * metricbeat             (HEALTHY)
	//                            Running
	//   * filebeat_monitoring    (HEALTHY)
	//                            Running
	//   * metricbeat_monitoring  (HEALTHY)
	//                            Running
}

func ExamplejsonOutput() {
	jsonOutput(os.Stdout, testStatus)
	// Output:
	// {
	//     "Status": 2,
	//     "Message": "",
	//     "Applications": [
	//         {
	//             "ID": "id_1",
	//             "Name": "filebeat",
	//             "Status": 2,
	//             "Message": "Running",
	//             "Payload": null
	//         },
	//         {
	//             "ID": "id_2",
	//             "Name": "metricbeat",
	//             "Status": 2,
	//             "Message": "Running",
	//             "Payload": null
	//         },
	//         {
	//             "ID": "id_3",
	//             "Name": "filebeat_monitoring",
	//             "Status": 2,
	//             "Message": "Running",
	//             "Payload": null
	//         },
	//         {
	//             "ID": "id_4",
	//             "Name": "metricbeat_monitoring",
	//             "Status": 2,
	//             "Message": "Running",
	//             "Payload": null
	//         }
	//     ]
	// }
}

func ExampleyamlOutput() {
	yamlOutput(os.Stdout, testStatus)
	// Output:
	// status: 2
	// message: ""
	// applications:
	// - id: id_1
	//   name: filebeat
	//   status: 2
	//   message: Running
	//   payload: {}
	// - id: id_2
	//   name: metricbeat
	//   status: 2
	//   message: Running
	//   payload: {}
	// - id: id_3
	//   name: filebeat_monitoring
	//   status: 2
	//   message: Running
	//   payload: {}
	// - id: id_4
	//   name: metricbeat_monitoring
	//   status: 2
	//   message: Running
	//   payload: {}
}
