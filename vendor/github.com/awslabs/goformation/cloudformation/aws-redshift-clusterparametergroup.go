package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSRedshiftClusterParameterGroup AWS CloudFormation Resource (AWS::Redshift::ClusterParameterGroup)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-redshift-clusterparametergroup.html
type AWSRedshiftClusterParameterGroup struct {

	// Description AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-redshift-clusterparametergroup.html#cfn-redshift-clusterparametergroup-description
	Description string `json:"Description,omitempty"`

	// ParameterGroupFamily AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-redshift-clusterparametergroup.html#cfn-redshift-clusterparametergroup-parametergroupfamily
	ParameterGroupFamily string `json:"ParameterGroupFamily,omitempty"`

	// Parameters AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-redshift-clusterparametergroup.html#cfn-redshift-clusterparametergroup-parameters
	Parameters []AWSRedshiftClusterParameterGroup_Parameter `json:"Parameters,omitempty"`

	// Tags AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-redshift-clusterparametergroup.html#cfn-redshift-clusterparametergroup-tags
	Tags []Tag `json:"Tags,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSRedshiftClusterParameterGroup) AWSCloudFormationType() string {
	return "AWS::Redshift::ClusterParameterGroup"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSRedshiftClusterParameterGroup) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSRedshiftClusterParameterGroup) MarshalJSON() ([]byte, error) {
	type Properties AWSRedshiftClusterParameterGroup
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
func (r *AWSRedshiftClusterParameterGroup) UnmarshalJSON(b []byte) error {
	type Properties AWSRedshiftClusterParameterGroup
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
		*r = AWSRedshiftClusterParameterGroup(*res.Properties)
	}

	return nil
}

// GetAllAWSRedshiftClusterParameterGroupResources retrieves all AWSRedshiftClusterParameterGroup items from an AWS CloudFormation template
func (t *Template) GetAllAWSRedshiftClusterParameterGroupResources() map[string]AWSRedshiftClusterParameterGroup {
	results := map[string]AWSRedshiftClusterParameterGroup{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSRedshiftClusterParameterGroup:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::Redshift::ClusterParameterGroup" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSRedshiftClusterParameterGroup
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

// GetAWSRedshiftClusterParameterGroupWithName retrieves all AWSRedshiftClusterParameterGroup items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSRedshiftClusterParameterGroupWithName(name string) (AWSRedshiftClusterParameterGroup, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSRedshiftClusterParameterGroup:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::Redshift::ClusterParameterGroup" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSRedshiftClusterParameterGroup
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSRedshiftClusterParameterGroup{}, errors.New("resource not found")
}
