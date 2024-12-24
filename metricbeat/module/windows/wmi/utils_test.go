package wmi

import (
	"fmt"
	"testing"
	"time"

	wmi "github.com/microsoft/wmi/pkg/wmiinstance"
	"github.com/stretchr/testify/assert"
)

type MockWmiSession struct {
}

const MockTimeout = time.Second * 5

// Mock Implementation of QueryInstances function
// This simulate a long-running query
func (c *MockWmiSession) QueryInstances(queryExpression string) ([]*wmi.WmiInstance, error) {
	time.Sleep(MockTimeout)
	return []*wmi.WmiInstance{}, nil
}

func TestExecuteGuardedQueryInstances(t *testing.T) {
	mockSession := new(MockWmiSession)
	query := "SELECT * FROM Win32_OpeartingSystem"
	timeout := 200 * time.Millisecond

	startTime := time.Now()
	expectedError := fmt.Errorf("the execution of the query'%s' exceeded the threshold of %s", query, timeout)
	_, err := ExecuteGuardedQueryInstances(mockSession, query, timeout)
	// Make sure the return time is less than the MockTimeout
	assert.Less(t, time.Since(startTime), MockTimeout, "The return time should be less than the sleep time")
	// Make sure the error returned is the expected one
	assert.Equal(t, err, expectedError, "Expected the returned error to match the expected error")
}
