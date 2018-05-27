package api

import (
	"net/http"

	uuid "github.com/satori/go.uuid"

	"github.com/elastic/beats/libbeat/common"
)

// Enroll a beat in central management, this call returns a valid access token to retrieve configurations
func (c *Client) Enroll(beatType, beatVersion, hostname string, beatUUID uuid.UUID, enrollmentToken string) (string, error) {
	params := common.MapStr{
		"type":      beatType,
		"host_name": hostname,
		"version":   beatVersion,
	}

	resp := struct {
		AccessToken string `json:"access_token"`
	}{}

	headers := http.Header{}
	headers.Set("kbn-beats-enrollment-token", enrollmentToken)

	_, err := c.request("POST", "/api/beats/agent/"+beatUUID.String(), params, headers, &resp)
	if err != nil {
		return "", err
	}

	return resp.AccessToken, err
}
