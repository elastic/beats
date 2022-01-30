package beater

import (
	"encoding/json"
	"github.com/elastic/beats/v7/kubebeat/config"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEvaluationResultParserParseResult(t *testing.T) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(jsonExample), &result)
	if err != nil {
		return
	}
	cycleId, _ := uuid.NewV4()
	index := config.Datastream("", config.ResultsDatastreamIndexPrefix)
	//Creating a new evaluation parser
	parser, _ := NewEvaluationResultParser(index)

	parsedResult, err := parser.ParseResult(result, cycleId)
	if err != nil {
		assert.Fail(t, "error during parsing of the json", err)
	}

	for _, event := range parsedResult {
		assert.Equal(t, cycleId, event.Fields["cycle_id"], "event cycle_id is not correct")
		assert.NotEmpty(t, event.Timestamp, `event timestamp is missing`)
		assert.NotEmpty(t, event.Fields["result"], "event result is missing")
		assert.NotEmpty(t, event.Fields["rule"], "event rule is missing")
		assert.NotEmpty(t, event.Fields["resource"], "event resource is missing")
	}
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
