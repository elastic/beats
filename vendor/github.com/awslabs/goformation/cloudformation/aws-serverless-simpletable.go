package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSServerlessSimpleTable AWS CloudFormation Resource (AWS::Serverless::SimpleTable)
// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#awsserverlesssimpletable
type AWSServerlessSimpleTable struct {

	// PrimaryKey AWS CloudFormation Property
	// Required: false
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#primary-key-object
	PrimaryKey *AWSServerlessSimpleTable_PrimaryKey `json:"PrimaryKey,omitempty"`

	// ProvisionedThroughput AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dynamodb-provisionedthroughput.html
	ProvisionedThroughput *AWSServerlessSimpleTable_ProvisionedThroughput `json:"ProvisionedThroughput,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSServerlessSimpleTable) AWSCloudFormationType() string {
	return "AWS::Serverless::SimpleTable"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSServerlessSimpleTable) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSServerlessSimpleTable) MarshalJSON() ([]byte, error) {
	type Properties AWSServerlessSimpleTable
	return json.Marshal(&struct {
		Type           string
		Properties     Properties
		DeletionPolicy DeletionPolicy `json:"DeletionPolicy,omitempty"`
	}{
		Type:           r.AWSCloudFormationType(),
		Properties:     (Properties)(r),
		DeletionPolicy: r._deletionPolicy,
	})
}

// UnmarshalJSON is a custom JSON unmarshalling hook that strips the outer
// AWS CloudFormation resource object, and just keeps the 'Properties' field.
func (r *AWSServerlessSimpleTable) UnmarshalJSON(b []byte) error {
	type Properties AWSServerlessSimpleTable
	res := &struct {
		Type       string
		Properties *Properties
	}{}
	if err := json.Unmarshal(b, &res); err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return err
	}

	// If the resource has no Properties set, it could be nil
	if res.Properties != nil {
		*r = AWSServerlessSimpleTable(*res.Properties)
	}

	return nil
}

// GetAllAWSServerlessSimpleTableResources retrieves all AWSServerlessSimpleTable items from an AWS CloudFormation template
func (t *Template) GetAllAWSServerlessSimpleTableResources() map[string]AWSServerlessSimpleTable {
	results := map[string]AWSServerlessSimpleTable{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSServerlessSimpleTable:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::Serverless::SimpleTable" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSServerlessSimpleTable
						if err := json.Unmarshal(b, &result); err == nil {
							results[name] = result
						}
					}
				}
			}
		}
	}
	return results
}

// GetAWSServerlessSimpleTableWithName retrieves all AWSServerlessSimpleTable items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSServerlessSimpleTableWithName(name string) (AWSServerlessSimpleTable, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSServerlessSimpleTable:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::Serverless::SimpleTable" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSServerlessSimpleTable
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSServerlessSimpleTable{}, errors.New("resource not found")
}
