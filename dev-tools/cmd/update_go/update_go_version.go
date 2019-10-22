package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func main() {
	currVersion := getGoVersion()
	newVersion := flag.String("newversion", currVersion, "new version of Go")

	flag.Parse()
	if flag.NFlag() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	updateGoVersion(currVersion, *newVersion)
}

func getGoVersion() string {
	version, err := ioutil.ReadFile(".go-version")
	checkErr(err)
	return (strings.TrimRight(string(version), "\r\n"))
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func updateGoVersion(oldVersion, newVersion string) {
	files := []string{".go-version",
		"auditbeat/Dockerfile",
		"filebeat/Dockerfile",
		"heartbeat/Dockerfile",
		"journalbeat/Dockerfile",
		"libbeat/docs/version.asciidoc",
		"metricbeat/Dockerfile",
		"x-pack/functionbeat/Dockerfile"}

	for _, file := range files {
		fmt.Printf("Updating Go version from %s to %s in %s\n", oldVersion, newVersion, file)
		content, err := ioutil.ReadFile(file)
		checkErr(err)
		updatedContent := strings.ReplaceAll(string(content), oldVersion, newVersion)
		ioutil.WriteFile(file, []byte(updatedContent), 0644)
		checkErr(err)
	}
}
