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

//go:build windows

package file_integrity

import (
	"fmt"
	"strings"
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"
	"golang.org/x/sys/windows"

	"github.com/elastic/elastic-agent-libs/logp"
)

// trieNode represents a node in the trie structure used to map device paths to drive letters.
// Each node can have children for the next character in the path, and terminal nodes
// store the complete drive path mapping.
type trieNode struct {
	children  map[rune]*trieNode
	drivePath string // Set only for terminal nodes that represent complete device paths
}

// deviceResolver converts Windows device paths (like \Device\HarddiskVolume1\path) to
// standard drive letter paths (like C:\path). It maintains a trie structure for efficient
// prefix matching and uses an LRU cache to speed up frequent translations.
//
// The resolver handles the complexity of Windows device path resolution by:
//   - Building a trie of device-to-drive mappings at startup
//   - Caching translation results for performance
//   - Thread-safe concurrent access
type deviceResolver struct {
	mu               sync.RWMutex
	root             *trieNode
	translationCache *lru.Cache[string, string]
	log              *logp.Logger
}

func newDeviceResolver(log *logp.Logger) (*deviceResolver, error) {
	cache, err := lru.New[string, string](1000) // Cache frequent translations
	if err != nil {
		return nil, fmt.Errorf("failed to create translation cache: %w", err)
	}

	resolver := &deviceResolver{
		root:             &trieNode{children: make(map[rune]*trieNode)},
		translationCache: cache,
		log:              log,
	}

	if err := resolver.buildDeviceMapping(); err != nil {
		return nil, fmt.Errorf("failed to build initial device mapping: %w", err)
	}

	return resolver, nil
}

func (dr *deviceResolver) translateDevicePath(kernelPath string) string {
	if kernelPath == "" {
		return ""
	}

	if cached, found := dr.translationCache.Get(kernelPath); found {
		return cached
	}

	dr.mu.RLock()
	result := dr.translateUsingTrie(kernelPath)
	dr.mu.RUnlock()

	dr.translationCache.Add(kernelPath, result)

	return result
}

// translateUsingTrie performs the actual trie-based translation
func (dr *deviceResolver) translateUsingTrie(kernelPath string) string {
	node := dr.root
	longestMatch := ""
	matchedDrive := ""

	// Walk through the path character by character
	for i, char := range kernelPath {
		if child, exists := node.children[char]; exists {
			node = child
			// If this node represents a complete device path, record it
			if node.drivePath != "" {
				longestMatch = kernelPath[:i+1]
				matchedDrive = node.drivePath
			}
		} else {
			break
		}
	}

	if longestMatch != "" {
		// Replace the device prefix with the drive letter
		remainder := kernelPath[len(longestMatch):]
		if strings.HasPrefix(remainder, "\\") {
			return matchedDrive + remainder
		} else if remainder == "" {
			return matchedDrive
		} else {
			return matchedDrive + "\\" + remainder
		}
	}

	return kernelPath // Return original if no match found
}

// buildDeviceMapping builds the device mapping and trie structure
func (dr *deviceResolver) buildDeviceMapping() error {
	deviceMap, err := dr.buildDeviceMap()
	if err != nil {
		return err
	}

	dr.mu.Lock()
	defer dr.mu.Unlock()

	dr.buildTrie(deviceMap)
	dr.log.Debugw("Device mapping built", "device_count", len(deviceMap))
	return nil
}

// buildTrie constructs the trie from the device mapping
func (dr *deviceResolver) buildTrie(deviceMap map[string]string) {
	dr.root = &trieNode{children: make(map[rune]*trieNode)}

	for devicePath, drivePath := range deviceMap {
		dr.insertIntoTrie(devicePath, drivePath)
	}
}

// insertIntoTrie inserts a device path into the trie
func (dr *deviceResolver) insertIntoTrie(devicePath, drivePath string) {
	node := dr.root
	for _, char := range devicePath {
		if node.children[char] == nil {
			node.children[char] = &trieNode{children: make(map[rune]*trieNode)}
		}
		node = node.children[char]
	}
	node.drivePath = drivePath
}

// buildDeviceMap builds the device-to-drive mapping using Windows API
func (dr *deviceResolver) buildDeviceMap() (map[string]string, error) {
	deviceMap := make(map[string]string)

	// Get logical drives bitmask
	bitmask, err := windows.GetLogicalDrives()
	if err != nil {
		return nil, fmt.Errorf("GetLogicalDrives failed: %w", err)
	}

	// Query each drive letter
	for i := 0; i < 26; i++ {
		if (bitmask>>i)&1 == 1 {
			driveLetter := string(byte('A' + i))
			drivePath := driveLetter + ":"

			// Query the device path for this drive
			buffer := make([]uint16, windows.MAX_PATH)
			_, err := windows.QueryDosDevice(
				windows.StringToUTF16Ptr(drivePath),
				&buffer[0],
				uint32(len(buffer)),
			)
			if err != nil {
				dr.log.Debugw("Failed to query device for drive",
					"drive", drivePath, "error", err)
				continue
			}

			devicePath := windows.UTF16ToString(buffer)
			if devicePath != "" {
				deviceMap[devicePath] = drivePath
			}
		}
	}

	return deviceMap, nil
}
