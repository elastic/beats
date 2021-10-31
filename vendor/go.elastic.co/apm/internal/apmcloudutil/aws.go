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

package apmcloudutil

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"go.elastic.co/apm/model"
)

const (
	ec2TokenURL    = "http://169.254.169.254/latest/api/token"
	ec2MetadataURL = "http://169.254.169.254/latest/dynamic/instance-identity/document"
)

// See: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/configuring-instance-metadata-service.html
func getAWSCloudMetadata(ctx context.Context, client *http.Client, out *model.Cloud) error {
	token, err := getAWSToken(ctx, client)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("GET", ec2MetadataURL, nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("X-aws-ec2-metadata-token", token)
	}

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}

	var ec2Metadata struct {
		AccountID        string `json:"accountId"`
		AvailabilityZone string `json:"availabilityZone"`
		Region           string `json:"region"`
		InstanceID       string `json:"instanceId"`
		InstanceType     string `json:"instanceType"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ec2Metadata); err != nil {
		return err
	}

	out.Region = ec2Metadata.Region
	out.AvailabilityZone = ec2Metadata.AvailabilityZone
	if ec2Metadata.InstanceID != "" {
		out.Instance = &model.CloudInstance{ID: ec2Metadata.InstanceID}
	}
	if ec2Metadata.InstanceType != "" {
		out.Machine = &model.CloudMachine{Type: ec2Metadata.InstanceType}
	}
	if ec2Metadata.AccountID != "" {
		out.Account = &model.CloudAccount{ID: ec2Metadata.AccountID}
	}
	return nil
}

func getAWSToken(ctx context.Context, client *http.Client) (string, error) {
	req, err := http.NewRequest("PUT", ec2TokenURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-aws-ec2-metadata-token-ttl-seconds", "300")
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	token, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(token), nil
}
