package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSRDSDBClusterParameterGroup AWS CloudFormation Resource (AWS::RDS::DBClusterParameterGroup)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-rds-dbclusterparametergroup.html
type AWSRDSDBClusterParameterGroup struct {

	// Description AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-rds-dbclusterparametergroup.html#cfn-rds-dbclusterparametergroup-description
	Description string `json:"Description,omitempty"`

	// Family AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-rds-dbclusterparametergroup.html#cfn-rds-dbclusterparametergroup-family
	Family string `json:"Family,omitempty"`

	// Parameters AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-rds-dbclusterparametergroup.html#cfn-rds-dbclusterparametergroup-parameters
	Parameters interface{} `json:"Parameters,omitempty"`

	// Tags AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-rds-dbclusterparametergroup.html#cfn-rds-dbclusterparametergroup-tags
	Tags []Tag `json:"Tags,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSRDSDBClusterParameterGroup) AWSCloudFormationType() string {
	return "AWS::RDS::DBClusterParameterGroup"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSRDSDBClusterParameterGroup) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSRDSDBClusterParameterGroup) MarshalJSON() ([]byte, error) {
	type Properties AWSRDSDBClusterParameterGroup
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
func (r *AWSRDSDBClusterParameterGroup) UnmarshalJSON(b []byte) error {
	type Properties AWSRDSDBClusterParameterGroup
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
		*r = AWSRDSDBClusterParameterGroup(*res.Properties)
	}

	return nil
}

// GetAllAWSRDSDBClusterParameterGroupResources retrieves all AWSRDSDBClusterParameterGroup items from an AWS CloudFormation template
func (t *Template) GetAllAWSRDSDBClusterParameterGroupResources() map[string]AWSRDSDBClusterParameterGroup {
	results := map[string]AWSRDSDBClusterParameterGroup{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSRDSDBClusterParameterGroup:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::RDS::DBClusterParameterGroup" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSRDSDBClusterParameterGroup
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

// GetAWSRDSDBClusterParameterGroupWithName retrieves all AWSRDSDBClusterParameterGroup items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSRDSDBClusterParameterGroupWithName(name string) (AWSRDSDBClusterParameterGroup, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSRDSDBClusterParameterGroup:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::RDS::DBClusterParameterGroup" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSRDSDBClusterParameterGroup
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSRDSDBClusterParameterGroup{}, errors.New("resource not found")
}
