package zookeeper

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"time"
)

// generic socket communication for four-letter commands.  returns a reader object to
// pass to a mapping that'll dissect the output
func RunCommand(command string, connectionString string, timeout time.Duration) (responseReader io.Reader, err error) {

	conn, err := net.DialTimeout("tcp", connectionString, timeout)
	if err != nil {
		return nil, fmt.Errorf("Error connecting to host (%s): %v", connectionString, err)
	}

	defer conn.Close()

	// Set read and write timeout
	err = conn.SetDeadline(time.Now().Add(timeout))
	if err != nil {
		return nil, err
	}

	// Write 4 letter command
	_, err = conn.Write([]byte(command))
	if err != nil {
		return nil, fmt.Errorf("Error writing %s command: %v", command, err)
	}

	// Read the data
	result, err := ioutil.ReadAll(conn)
	if err != nil {
		return nil, fmt.Errorf("Error reading %s command: %v", command, err)
	}

	return bytes.NewReader(result), nil
}
