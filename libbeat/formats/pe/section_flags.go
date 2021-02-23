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

package pe

const (
	IMAGE_SCN_TYPE_NO_PAD            uint32 = 0x00000008
	IMAGE_SCN_CNT_CODE               uint32 = 0x00000020
	IMAGE_SCN_CNT_INITIALIZED_DATA   uint32 = 0x00000040
	IMAGE_SCN_CNT_UNINITIALIZED_DATA uint32 = 0x00000080
	IMAGE_SCN_LNK_OTHER              uint32 = 0x00000100
	IMAGE_SCN_LNK_INFO               uint32 = 0x00000200
	IMAGE_SCN_LNK_REMOVE             uint32 = 0x00000800
	IMAGE_SCN_LNK_COMDAT             uint32 = 0x00001000
	IMAGE_SCN_GPREL                  uint32 = 0x00008000
	IMAGE_SCN_MEM_PURGEABLE          uint32 = 0x00020000
	IMAGE_SCN_MEM_16BIT              uint32 = 0x00020000
	IMAGE_SCN_MEM_LOCKED             uint32 = 0x00040000
	IMAGE_SCN_MEM_PRELOAD            uint32 = 0x00080000
	IMAGE_SCN_ALIGN_1BYTES           uint32 = 0x00100000
	IMAGE_SCN_ALIGN_2BYTES           uint32 = 0x00200000
	IMAGE_SCN_ALIGN_4BYTES           uint32 = 0x00300000
	IMAGE_SCN_ALIGN_8BYTES           uint32 = 0x00400000
	IMAGE_SCN_ALIGN_16BYTES          uint32 = 0x00500000
	IMAGE_SCN_ALIGN_32BYTES          uint32 = 0x00600000
	IMAGE_SCN_ALIGN_64BYTES          uint32 = 0x00700000
	IMAGE_SCN_ALIGN_128BYTES         uint32 = 0x00800000
	IMAGE_SCN_ALIGN_256BYTES         uint32 = 0x00900000
	IMAGE_SCN_ALIGN_512BYTES         uint32 = 0x00A00000
	IMAGE_SCN_ALIGN_1024BYTES        uint32 = 0x00B00000
	IMAGE_SCN_ALIGN_2048BYTES        uint32 = 0x00C00000
	IMAGE_SCN_ALIGN_4096BYTES        uint32 = 0x00D00000
	IMAGE_SCN_ALIGN_8192BYTES        uint32 = 0x00E00000
	IMAGE_SCN_LNK_NRELOC_OVFL        uint32 = 0x01000000
	IMAGE_SCN_MEM_DISCARDABLE        uint32 = 0x02000000
	IMAGE_SCN_MEM_NOT_CACHED         uint32 = 0x04000000
	IMAGE_SCN_MEM_NOT_PAGED          uint32 = 0x08000000
	IMAGE_SCN_MEM_SHARED             uint32 = 0x10000000
	IMAGE_SCN_MEM_EXECUTE            uint32 = 0x20000000
	IMAGE_SCN_MEM_READ               uint32 = 0x40000000
	IMAGE_SCN_MEM_WRITE              uint32 = 0x80000000
)

func translateSectionFlags(characteristics uint32) []string {
	flags := []string{}
	if (characteristics & IMAGE_SCN_TYPE_NO_PAD) != 0 {
		flags = append(flags, "IMAGE_SCN_TYPE_NO_PAD")
	}
	if (characteristics & IMAGE_SCN_CNT_CODE) != 0 {
		flags = append(flags, "IMAGE_SCN_CNT_CODE")
	}
	if (characteristics & IMAGE_SCN_CNT_INITIALIZED_DATA) != 0 {
		flags = append(flags, "IMAGE_SCN_CNT_INITIALIZED_DATA")
	}
	if (characteristics & IMAGE_SCN_CNT_UNINITIALIZED_DATA) != 0 {
		flags = append(flags, "IMAGE_SCN_CNT_UNINITIALIZED_DATA")
	}
	if (characteristics & IMAGE_SCN_LNK_OTHER) != 0 {
		flags = append(flags, "IMAGE_SCN_LNK_OTHER")
	}
	if (characteristics & IMAGE_SCN_LNK_INFO) != 0 {
		flags = append(flags, "IMAGE_SCN_LNK_INFO")
	}
	if (characteristics & IMAGE_SCN_LNK_REMOVE) != 0 {
		flags = append(flags, "IMAGE_SCN_LNK_REMOVE")
	}
	if (characteristics & IMAGE_SCN_LNK_COMDAT) != 0 {
		flags = append(flags, "IMAGE_SCN_LNK_COMDAT")
	}
	if (characteristics & IMAGE_SCN_GPREL) != 0 {
		flags = append(flags, "IMAGE_SCN_GPREL")
	}
	if (characteristics & IMAGE_SCN_MEM_PURGEABLE) != 0 {
		flags = append(flags, "IMAGE_SCN_MEM_PURGEABLE")
	}
	if (characteristics & IMAGE_SCN_MEM_16BIT) != 0 {
		flags = append(flags, "IMAGE_SCN_MEM_16BIT")
	}
	if (characteristics & IMAGE_SCN_MEM_LOCKED) != 0 {
		flags = append(flags, "IMAGE_SCN_MEM_LOCKED")
	}
	if (characteristics & IMAGE_SCN_MEM_PRELOAD) != 0 {
		flags = append(flags, "IMAGE_SCN_MEM_PRELOAD")
	}
	if (characteristics & IMAGE_SCN_ALIGN_1BYTES) != 0 {
		flags = append(flags, "IMAGE_SCN_ALIGN_1BYTES")
	}
	if (characteristics & IMAGE_SCN_ALIGN_2BYTES) != 0 {
		flags = append(flags, "IMAGE_SCN_ALIGN_2BYTES")
	}
	if (characteristics & IMAGE_SCN_ALIGN_4BYTES) != 0 {
		flags = append(flags, "IMAGE_SCN_ALIGN_4BYTES")
	}
	if (characteristics & IMAGE_SCN_ALIGN_8BYTES) != 0 {
		flags = append(flags, "IMAGE_SCN_ALIGN_8BYTES")
	}
	if (characteristics & IMAGE_SCN_ALIGN_16BYTES) != 0 {
		flags = append(flags, "IMAGE_SCN_ALIGN_16BYTES")
	}
	if (characteristics & IMAGE_SCN_ALIGN_32BYTES) != 0 {
		flags = append(flags, "IMAGE_SCN_ALIGN_32BYTES")
	}
	if (characteristics & IMAGE_SCN_ALIGN_64BYTES) != 0 {
		flags = append(flags, "IMAGE_SCN_ALIGN_64BYTES")
	}
	if (characteristics & IMAGE_SCN_ALIGN_128BYTES) != 0 {
		flags = append(flags, "IMAGE_SCN_ALIGN_128BYTES")
	}
	if (characteristics & IMAGE_SCN_ALIGN_256BYTES) != 0 {
		flags = append(flags, "IMAGE_SCN_ALIGN_256BYTES")
	}
	if (characteristics & IMAGE_SCN_ALIGN_512BYTES) != 0 {
		flags = append(flags, "IMAGE_SCN_ALIGN_512BYTES")
	}
	if (characteristics & IMAGE_SCN_ALIGN_1024BYTES) != 0 {
		flags = append(flags, "IMAGE_SCN_ALIGN_1024BYTES")
	}
	if (characteristics & IMAGE_SCN_ALIGN_2048BYTES) != 0 {
		flags = append(flags, "IMAGE_SCN_ALIGN_2048BYTES")
	}
	if (characteristics & IMAGE_SCN_ALIGN_4096BYTES) != 0 {
		flags = append(flags, "IMAGE_SCN_ALIGN_4096BYTES")
	}
	if (characteristics & IMAGE_SCN_ALIGN_8192BYTES) != 0 {
		flags = append(flags, "IMAGE_SCN_ALIGN_8192BYTES")
	}
	if (characteristics & IMAGE_SCN_LNK_NRELOC_OVFL) != 0 {
		flags = append(flags, "IMAGE_SCN_LNK_NRELOC_OVFL")
	}
	if (characteristics & IMAGE_SCN_MEM_DISCARDABLE) != 0 {
		flags = append(flags, "IMAGE_SCN_MEM_DISCARDABLE")
	}
	if (characteristics & IMAGE_SCN_MEM_NOT_CACHED) != 0 {
		flags = append(flags, "IMAGE_SCN_MEM_NOT_CACHED")
	}
	if (characteristics & IMAGE_SCN_MEM_NOT_PAGED) != 0 {
		flags = append(flags, "IMAGE_SCN_MEM_NOT_PAGED")
	}
	if (characteristics & IMAGE_SCN_MEM_SHARED) != 0 {
		flags = append(flags, "IMAGE_SCN_MEM_SHARED")
	}
	if (characteristics & IMAGE_SCN_MEM_EXECUTE) != 0 {
		flags = append(flags, "IMAGE_SCN_MEM_EXECUTE")
	}
	if (characteristics & IMAGE_SCN_MEM_READ) != 0 {
		flags = append(flags, "IMAGE_SCN_MEM_READ")
	}
	if (characteristics & IMAGE_SCN_MEM_WRITE) != 0 {
		flags = append(flags, "IMAGE_SCN_MEM_WRITE")
	}
	return flags
}
