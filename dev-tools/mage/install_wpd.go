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

package mage

import (
	"fmt"
	"os"
)

const (
	wpdPackVer = "4_1_2"
	wpdPackUrl = "https://www.winpcap.org/install/bin/WpdPack_%s.zip"
)

// InstallWpd installs WinPcap. Some tests on old branches e.g. x-pack/filebeat or building packebeat
// require the Winpcap Developer Pack which provides the necessary headers and libs.
// google/gopacket expects to find this in C:/WpdPack/.
// see https://elastic.slack.com/archives/C0522G6FBNE/p1708356267963779?thread_ts=1708356249.822359&cid=C0522G6FBNE
func InstallWpd() {
	homeDir, _ := os.UserHomeDir()
	url := fmt.Sprintf(wpdPackUrl, wpdPackVer)
	downloadPath := fmt.Sprintf("%s/%s", homeDir, wpdPackVer)

	fmt.Println("--- Downloading WPD Pack")
	file, err := DownloadFile(url, downloadPath)
	if err != nil {
		panic("Error downloading WpdPack: " + err.Error())
	}

	fmt.Println("--- Extracting WPD Pack")
	err = Extract(file, "C:/")
	if err != nil {
		panic("Error extracting the archive: " + err.Error())
	}
}
