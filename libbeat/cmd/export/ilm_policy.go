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

package export

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/libbeat/cmd/instance"
)

// GenGetILMPolicyCmd is the command used to export the ilm policy.
func GenGetILMPolicyCmd() *cobra.Command {
	genTemplateConfigCmd := &cobra.Command{
		Use:   "ilm-policy",
		Short: "Export ILM policy",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(instance.ILMPolicy.StringToPrint())
		},
	}

	return genTemplateConfigCmd
}
