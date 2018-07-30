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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/ewgRa/ci-utils/src/diff_liner"
	"github.com/ewgRa/gocsfixer"
	"github.com/google/go-github/github"
)

func main() {
	prLiner := flag.String("pr-liner", "", "Pull request liner")
	csFixerCommentsFile := flag.String("csfixer-comments", "", "Cs fixer comments")

	flag.Parse()

	if *prLiner == "" || *csFixerCommentsFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	linerResp := diff_liner.ReadLinerResponse(*prLiner)

	var comments []*github.PullRequestComment

	csFixerComments, err := gocsfixer.ReadResults(*csFixerCommentsFile)
	if err != nil {
		panic(err)
	}

	for _, fixerComment := range csFixerComments {
		prLine := linerResp.GetDiffLine(fixerComment.File, fixerComment.Line)

		if prLine == 0 {
			continue
		}

		body := "[" + fixerComment.Type + "] " + fixerComment.Text

		cmd := exec.Command("git", "log", "--pretty=format:%H", "-1", fixerComment.File)
		output, err := cmd.Output()
		if err != nil {
			panic(err)
		}

		commit := string(output)

		comments = append(comments, &github.PullRequestComment{
			Body:     &body,
			CommitID: &commit,
			Path:     &fixerComment.File,
			Position: &prLine,
		})

	}

	if len(comments) > 0 {
		jsonData, err := json.Marshal(comments)
		if err != nil {
			panic(err)
		}

		fmt.Println(string(jsonData))
	} else {
		fmt.Println("[]")
	}
}
