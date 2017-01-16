package ceph

import (
	"bytes"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/fsouza/go-dockerclient"
	"os"
	"strings"
)

func ExecInContainer(cmd []string) string {

	// create client
	endpoint := os.Getenv("DOCKER_HOST")
	if endpoint == "" {
		endpoint = "unix:///var/run/docker.sock"
	}
	client, err := docker.NewClient(endpoint)
	if err != nil {
		logp.Warn(err.Error())
	}

	// options to exec
	de := docker.CreateExecOptions{
		AttachStderr: true,
		AttachStdin:  true,
		AttachStdout: true,
		Tty:          false,
		Cmd:          cmd,
		Container:    "metricbeatceph",
	}

	var (
		dExec          *docker.Exec
		stdout, stderr bytes.Buffer
	)

	if dExec, err = client.CreateExec(de); err != nil {
		logp.Warn("CreateExec Error: %s", err)
	} else {

		// exec command
		var reader = strings.NewReader("send value")

		execId := dExec.ID

		opts := docker.StartExecOptions{
			OutputStream: &stdout,
			ErrorStream:  &stderr,
			InputStream:  reader,
			RawTerminal:  false,
		}

		if err = client.StartExec(execId, opts); err != nil {
			logp.Warn("StartExec Error: %s", err)
		}
	}

	return stdout.String()
}
