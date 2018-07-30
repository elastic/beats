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
