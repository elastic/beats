// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package cmd

import (
	"fmt"
	"log"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/packetbeat/sniffer"
)

func genDevicesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "devices",
		Short: "List available devices",
		Run: func(cmd *cobra.Command, args []string) {
			printDevicesList()
		},
	}
}

func printDevicesList() {
	lst, err := sniffer.ListDeviceNames(true, true)
	if err != nil {
		log.Fatalf("Error getting devices list: %v\n", err)
	}

	if len(lst) == 0 {
		fmt.Printf("No devices found.")
		if runtime.GOOS != "windows" {
			fmt.Println(" You might need sudo?")
		} else {
			fmt.Println("")
		}
	}

	for i, d := range lst {
		fmt.Printf("%d: %s\n", i, d)
	}
}
