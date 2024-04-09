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

package kubernetes

type Summary struct {
	Node struct {
		CPU struct {
			Time                 string `json:"time"`
			UsageCoreNanoSeconds uint64 `json:"usageCoreNanoSeconds"`
			UsageNanoCores       uint64 `json:"usageNanoCores"`
		} `json:"cpu"`
		Fs struct {
			AvailableBytes uint64 `json:"availableBytes"`
			CapacityBytes  uint64 `json:"capacityBytes"`
			Inodes         uint64 `json:"inodes"`
			InodesFree     uint64 `json:"inodesFree"`
			InodesUsed     uint64 `json:"inodesUsed"`
			UsedBytes      uint64 `json:"usedBytes"`
		} `json:"fs"`
		Memory struct {
			AvailableBytes  uint64 `json:"availableBytes"`
			MajorPageFaults uint64 `json:"majorPageFaults"`
			PageFaults      uint64 `json:"pageFaults"`
			RssBytes        uint64 `json:"rssBytes"`
			Time            string `json:"time"`
			UsageBytes      uint64 `json:"usageBytes"`
			WorkingSetBytes uint64 `json:"workingSetBytes"`
		} `json:"memory"`
		Network struct {
			RxBytes  uint64 `json:"rxBytes"`
			RxErrors uint64 `json:"rxErrors"`
			Time     string `json:"time"`
			TxBytes  uint64 `json:"txBytes"`
			TxErrors uint64 `json:"txErrors"`
		} `json:"network"`
		NodeName string `json:"nodeName"`
		Runtime  struct {
			ImageFs struct {
				AvailableBytes uint64 `json:"availableBytes"`
				CapacityBytes  uint64 `json:"capacityBytes"`
				UsedBytes      uint64 `json:"usedBytes"`
			} `json:"imageFs"`
		} `json:"runtime"`
		StartTime        string `json:"startTime"`
		SystemContainers []struct {
			CPU struct {
				Time                 string `json:"time"`
				UsageCoreNanoSeconds uint64 `json:"usageCoreNanoSeconds"`
				UsageNanoCores       uint64 `json:"usageNanoCores"`
			} `json:"cpu"`
			Memory struct {
				MajorPageFaults uint64 `json:"majorPageFaults"`
				PageFaults      uint64 `json:"pageFaults"`
				RssBytes        uint64 `json:"rssBytes"`
				Time            string `json:"time"`
				UsageBytes      uint64 `json:"usageBytes"`
				WorkingSetBytes uint64 `json:"workingSetBytes"`
			} `json:"memory"`
			Name               string      `json:"name"`
			StartTime          string      `json:"startTime"`
			UserDefinedMetrics interface{} `json:"userDefinedMetrics"`
		} `json:"systemContainers"`
	} `json:"node"`
	Pods []struct {
		Containers []struct {
			CPU struct {
				Time                 string `json:"time"`
				UsageCoreNanoSeconds uint64 `json:"usageCoreNanoSeconds"`
				UsageNanoCores       uint64 `json:"usageNanoCores"`
			} `json:"cpu"`
			Logs struct {
				AvailableBytes uint64 `json:"availableBytes"`
				CapacityBytes  uint64 `json:"capacityBytes"`
				Inodes         uint64 `json:"inodes"`
				InodesFree     uint64 `json:"inodesFree"`
				InodesUsed     uint64 `json:"inodesUsed"`
				UsedBytes      uint64 `json:"usedBytes"`
			} `json:"logs"`
			Memory struct {
				AvailableBytes  uint64 `json:"availableBytes"`
				MajorPageFaults uint64 `json:"majorPageFaults"`
				PageFaults      uint64 `json:"pageFaults"`
				RssBytes        uint64 `json:"rssBytes"`
				Time            string `json:"time"`
				UsageBytes      uint64 `json:"usageBytes"`
				WorkingSetBytes uint64 `json:"workingSetBytes"`
			} `json:"memory"`
			Name   string `json:"name"`
			Rootfs struct {
				AvailableBytes uint64 `json:"availableBytes"`
				CapacityBytes  uint64 `json:"capacityBytes"`
				InodesUsed     uint64 `json:"inodesUsed"`
				UsedBytes      uint64 `json:"usedBytes"`
			} `json:"rootfs"`
			StartTime          string      `json:"startTime"`
			UserDefinedMetrics interface{} `json:"userDefinedMetrics"`
		} `json:"containers"`
		Network struct {
			RxBytes  uint64 `json:"rxBytes"`
			RxErrors uint64 `json:"rxErrors"`
			Time     string `json:"time"`
			TxBytes  uint64 `json:"txBytes"`
			TxErrors uint64 `json:"txErrors"`
		} `json:"network"`
		PodRef struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
			UID       string `json:"uid"`
		} `json:"podRef"`
		StartTime string `json:"startTime"`
		Volume    []struct {
			AvailableBytes uint64 `json:"availableBytes"`
			CapacityBytes  uint64 `json:"capacityBytes"`
			Inodes         uint64 `json:"inodes"`
			InodesFree     uint64 `json:"inodesFree"`
			InodesUsed     uint64 `json:"inodesUsed"`
			Name           string `json:"name"`
			UsedBytes      uint64 `json:"usedBytes"`
			PvcRef         struct {
				Name      string `json:"name"`
				Namespace string `json:"namespace"`
			} `json:"pvcRef"`
		} `json:"volume"`
	} `json:"pods"`
}
