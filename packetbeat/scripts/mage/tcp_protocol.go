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
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CreateTcpProtocol scaffolds a new packetbeat TCP protocol plugin from
// the templates in scriptsDir/tcp-protocol/{protocol}/.
// Replaces packetbeat/scripts/create_tcp_protocol.py.
func CreateTcpProtocol(scriptsDir, protocolName string) error {
	if protocolName == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Protocol Name [exampletcp]: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			protocolName = "exampletcp"
		} else {
			protocolName = input
		}
	}
	protocolName = strings.ToLower(protocolName)
	pluginType := protocolName + "Plugin"
	pluginVar := string(protocolName[0]) + "p"

	templateDir := filepath.Join(scriptsDir, "tcp-protocol", "{protocol}")

	return filepath.Walk(templateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		content := replaceTcpVars(string(data), protocolName, pluginType, pluginVar)
		newPath := replaceTcpVars(path, protocolName, pluginType, pluginVar)
		newPath = strings.ReplaceAll(newPath, ".go.tmpl", ".go")

		relPath := strings.TrimPrefix(newPath, scriptsDir+string(filepath.Separator)+"tcp-protocol"+string(filepath.Separator))

		writePath := filepath.Join("protos", relPath)

		dir := filepath.Dir(writePath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}

		if err := os.WriteFile(writePath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", writePath, err)
		}

		fmt.Printf("Created %s\n", writePath)
		return nil
	})
}

func replaceTcpVars(s, protocol, pluginType, pluginVar string) string {
	s = strings.ReplaceAll(s, "{protocol}", protocol)
	s = strings.ReplaceAll(s, "{plugin_var}", pluginVar)
	s = strings.ReplaceAll(s, "{plugin_type}", pluginType)
	return s
}
