package beater

import (
	"encoding/json"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestEvaluationResultParserParseResult(t *testing.T) {

	parser, _ := NewEvaluationResultParser()
	jsonResponse := jsonExample
	runId, _ := uuid.NewV4()
	timestamp := time.Now()
	var result map[string]interface{}
	json.Unmarshal([]byte(jsonResponse), &result)

	parsedResult, err := parser.ParseResult(result, runId, timestamp)
	if err != nil {
		assert.Fail(t, "error during parsing of the json", err)
	}

	// Assert first event
	firstEvent := parsedResult[0]
	assert.Equal(t, timestamp, firstEvent.Timestamp, "event timestamp is not correct")
	assert.Equal(t, runId, firstEvent.Fields["run_id"], "uid is not correct")
	assert.Equal(t, runId, firstEvent.Fields["resource"], "uid is not correct")
	assert.Equal(t, runId, firstEvent.Fields["rule"], "uid is not correct")

	// Assert second event
	secondEvent := parsedResult[0]
	assert.Equal(t, timestamp, secondEvent.Timestamp, "event timestamp is not correct")
	assert.Equal(t, runId, secondEvent.Fields["run_id"], "uid is not correct")
	assert.Equal(t, runId, secondEvent.Fields["resource"], "uid is not correct")
	assert.Equal(t, runId, secondEvent.Fields["rule"], "uid is not correct")

}

var jsonExample = `{
"findings":
[
{
	"result": {
	"evaluation": "failed",
	"evidence": {
		"filemode": "700"
	}
},
"rule": {
"benchmark": "CIS Kubernetes",
"description": "The scheduler.conf file is the kubeconfig file for the Scheduler. You should restrict its file permissions to maintain the integrity of the file. The file should be writable by only the administrators on the system.",
"impact": "None",
"name": "Ensure that the scheduler.conf file permissions are set to 644 or more restrictive",
"remediation": "chmod 644 /etc/kubernetes/scheduler.conf",
"tags": [
"CIS",
"CIS v1.6.0",
"Kubernetes",
"CIS 1.1.15",
"Master Node Configuration"
]
}
},
{
"result": {
"evaluation": "passed",
"evidence": {
"gid": "root",
"uid": "root"
}
},
"rule": {
"benchmark": "CIS Kubernetes",
"description": "The scheduler.conf file is the kubeconfig file for the Scheduler. You should set its file ownership to maintain the integrity of the file. The file should be owned by root:root.",
"impact": "None",
"name": "Ensure that the scheduler.conf file ownership is set to root:root",
"remediation": "chown root:root /etc/kubernetes/scheduler.conf",
"tags": [
"CIS",
"CIS v1.6.0",
"Kubernetes",
"CIS 1.1.16",
"Master Node Configuration"
]
}
}
],
"resource": {
"filename": "scheduler.conf",
"gid": "root",
"mode": "700",
"path": "/hostfs/etc/kubernetes/scheduler.conf",
"type": "file-system",
"uid": "root"
}
}
`
