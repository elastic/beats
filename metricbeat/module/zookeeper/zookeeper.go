/*
Package zookeeper is a Metricbeat module for ZooKeeper servers.
*/
package zookeeper

import (
	"bytes"
	"io"
	"io/ioutil"
	"net"
	"time"

	"github.com/pkg/errors"
)

// RunCommand establishes a TCP connection the ZooKeeper command port that
// accepts the four-letter ZooKeeper commands and sends the given command. It
// reads all response data received on the socket and returns an io.Reader
// containing that data.
func RunCommand(command, address string, timeout time.Duration) (io.Reader, error) {
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return nil, errors.Wrapf(err, "connection to host '%s' failed", address)
	}
	defer conn.Close()

	// Set read and write timeout.
	err = conn.SetDeadline(time.Now().Add(timeout))
	if err != nil {
		return nil, err
	}

	// Write four-letter command.
	_, err = conn.Write([]byte(command))
	if err != nil {
		return nil, errors.Wrapf(err, "writing command '%s' failed", command)
	}

	result, err := ioutil.ReadAll(conn)
	if err != nil {
		return nil, errors.Wrap(err, "read failed")
	}

	return bytes.NewReader(result), nil
}
