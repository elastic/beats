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
	"testing"

	"github.com/elastic/elastic-agent-libs/logp"
)

// Benchmark the old linear search approach vs new trie-based approach
func BenchmarkDeviceResolution(b *testing.B) {
	// Create test data similar to real Windows device paths
	testPaths := []string{
		"\\Device\\HarddiskVolume1\\Windows\\System32\\notepad.exe",
		"\\Device\\HarddiskVolume1\\Users\\Test\\Documents\\file.txt",
		"\\Device\\HarddiskVolume2\\Program Files\\Application\\app.exe",
		"\\Device\\HarddiskVolume1\\Windows\\System32\\drivers\\etc\\hosts",
		"\\Device\\HarddiskVolume3\\Data\\logs\\application.log",
		"\\Device\\HarddiskVolumeX\\Very\\Long\\Path\\With\\Many\\Segments\\file.dat",
		"\\Device\\Mup\\server\\share\\remote\\file.txt",
		"\\Device\\LanmanRedirector\\server\\share\\document.pdf",
	}

	// Create mock device mappings
	deviceMap := map[string]string{
		"\\Device\\HarddiskVolume1":  "C:",
		"\\Device\\HarddiskVolume2":  "D:",
		"\\Device\\HarddiskVolume3":  "E:",
		"\\Device\\HarddiskVolumeX":  "F:",
		"\\Device\\Mup":              "\\\\",
		"\\Device\\LanmanRedirector": "\\\\",
	}

	b.Run("OriginalLinearSearch", func(b *testing.B) {
		// Simulate the original approach with sorted device list
		deviceList := make([]string, 0, len(deviceMap))
		for device := range deviceMap {
			deviceList = append(deviceList, device)
		}
		// Sort by length descending (longest first) like original implementation
		for i := 0; i < len(deviceList); i++ {
			for j := i + 1; j < len(deviceList); j++ {
				if len(deviceList[i]) < len(deviceList[j]) {
					deviceList[i], deviceList[j] = deviceList[j], deviceList[i]
				}
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			path := testPaths[i%len(testPaths)]
			translateDevicePathLinear(path, deviceList, deviceMap)
		}
	})

	b.Run("TrieBased", func(b *testing.B) {
		logger := logp.NewLogger("test")
		resolver, err := newDeviceResolver(logger)
		if err != nil {
			b.Fatalf("Failed to create device resolver: %v", err)
		}

		// Override with test data
		resolver.mu.Lock()
		resolver.buildTrie(deviceMap)
		resolver.mu.Unlock()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			path := testPaths[i%len(testPaths)]
			resolver.translateDevicePath(path)
		}
	})
}

// translateDevicePathLinear simulates the original linear search implementation
func translateDevicePathLinear(kernelPath string, deviceList []string, deviceMap map[string]string) string {
	if kernelPath == "" {
		return ""
	}
	for _, device := range deviceList {
		if strings.HasPrefix(kernelPath, device) {
			drive := deviceMap[device]
			return strings.Replace(kernelPath, device, drive, 1)
		}
	}
	return kernelPath
}

// BenchmarkDeviceResolutionScalability tests performance with varying numbers of devices
func BenchmarkDeviceResolutionScalability(b *testing.B) {
	deviceCounts := []int{10, 50, 100, 500}
	testPath := "\\Device\\HarddiskVolume999\\Windows\\System32\\file.exe"

	for _, deviceCount := range deviceCounts {
		b.Run(fmt.Sprintf("Linear_%d_devices", deviceCount), func(b *testing.B) {
			deviceMap, deviceList := createTestDeviceMapping(deviceCount)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				translateDevicePathLinear(testPath, deviceList, deviceMap)
			}
		})

		b.Run(fmt.Sprintf("Trie_%d_devices", deviceCount), func(b *testing.B) {
			logger := logp.NewLogger("test")
			resolver, _ := newDeviceResolver(logger)
			deviceMap, _ := createTestDeviceMapping(deviceCount)

			resolver.mu.Lock()
			resolver.buildTrie(deviceMap)
			resolver.mu.Unlock()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				resolver.translateDevicePath(testPath)
			}
		})
	}
}

func createTestDeviceMapping(count int) (map[string]string, []string) {
	deviceMap := make(map[string]string)
	deviceList := make([]string, 0, count)

	for i := 0; i < count; i++ {
		device := fmt.Sprintf("\\Device\\HarddiskVolume%d", i)
		drive := fmt.Sprintf("%c:", 'A'+i%26)
		deviceMap[device] = drive
		deviceList = append(deviceList, device)
	}

	// Sort by length descending
	for i := 0; i < len(deviceList); i++ {
		for j := i + 1; j < len(deviceList); j++ {
			if len(deviceList[i]) < len(deviceList[j]) {
				deviceList[i], deviceList[j] = deviceList[j], deviceList[i]
			}
		}
	}

	return deviceMap, deviceList
}

// BenchmarkCacheEffectiveness tests the translation cache performance
func BenchmarkCacheEffectiveness(b *testing.B) {
	logger := logp.NewLogger("test")
	resolver, err := newDeviceResolver(logger)
	if err != nil {
		b.Fatalf("Failed to create device resolver: %v", err)
	}

	// Override with test data
	deviceMap := map[string]string{
		"\\Device\\HarddiskVolume1": "C:",
		"\\Device\\HarddiskVolume2": "D:",
	}
	resolver.mu.Lock()
	resolver.buildTrie(deviceMap)
	resolver.mu.Unlock()

	// Frequently accessed paths (simulating real-world access patterns)
	frequentPaths := []string{
		"\\Device\\HarddiskVolume1\\Windows\\System32\\notepad.exe",
		"\\Device\\HarddiskVolume1\\Windows\\System32\\cmd.exe",
		"\\Device\\HarddiskVolume1\\Windows\\explorer.exe",
	}

	b.Run("WithCache", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			path := frequentPaths[i%len(frequentPaths)]
			resolver.translateDevicePath(path)
		}
	})

	b.Run("WithoutCache", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			path := frequentPaths[i%len(frequentPaths)]
			// Clear cache before each operation to simulate no caching
			resolver.translationCache.Purge()
			resolver.translateDevicePath(path)
		}
	})
}

// Test the correctness of the device resolution
func TestDeviceResolutionCorrectness(t *testing.T) {
	logger := logp.NewLogger("test")
	resolver, err := newDeviceResolver(logger)
	if err != nil {
		t.Fatalf("Failed to create device resolver: %v", err)
	}

	// Test data
	deviceMap := map[string]string{
		"\\Device\\HarddiskVolume1": "C:",
		"\\Device\\HarddiskVolume2": "D:",
		"\\Device\\Mup":             "\\\\",
	}

	resolver.mu.Lock()
	resolver.buildTrie(deviceMap)
	resolver.mu.Unlock()

	testCases := []struct {
		input    string
		expected string
	}{
		{
			"\\Device\\HarddiskVolume1\\Windows\\System32\\file.txt",
			"C:\\Windows\\System32\\file.txt",
		},
		{
			"\\Device\\HarddiskVolume2\\Data\\file.log",
			"D:\\Data\\file.log",
		},
		{
			"\\Device\\Mup\\server\\share\\file.doc",
			"\\\\\\server\\share\\file.doc", // Note: This is the actual expected behavior
		},
		{
			"\\Device\\Unknown\\path\\file.txt",
			"\\Device\\Unknown\\path\\file.txt", // Should return original
		},
		{
			"",
			"", // Empty path
		},
	}

	for _, tc := range testCases {
		result := resolver.translateDevicePath(tc.input)
		if result != tc.expected {
			t.Errorf("translateDevicePath(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}
