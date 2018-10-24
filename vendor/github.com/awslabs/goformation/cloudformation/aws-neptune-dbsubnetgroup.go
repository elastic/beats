package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSNeptuneDBSubnetGroup AWS CloudFormation Resource (AWS::Neptune::DBSubnetGroup)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-neptune-dbsubnetgroup.html
type AWSNeptuneDBSubnetGroup struct {

	// DBSubnetGroupDescription AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-neptune-dbsubnetgroup.html#cfn-neptune-dbsubnetgroup-dbsubnetgroupdescription
	DBSubnetGroupDescription string `json:"DBSubnetGroupDescription,omitempty"`

	// DBSubnetGroupName AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-neptune-dbsubnetgroup.html#cfn-neptune-dbsubnetgroup-dbsubnetgroupname
	DBSubnetGroupName string `json:"DBSubnetGroupName,omitempty"`

	// SubnetIds AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-neptune-dbsubnetgroup.html#cfn-neptune-dbsubnetgroup-subnetids
	SubnetIds []string `json:"SubnetIds,omitempty"`

	// Tags AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-neptune-dbsubnetgroup.html#cfn-neptune-dbsubnetgroup-tags
	Tags []Tag `json:"Tags,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSNeptuneDBSubnetGroup) AWSCloudFormationType() string {
	return "AWS::Neptune::DBSubnetGroup"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSNeptuneDBSubnetGroup) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSNeptuneDBSubnetGroup) MarshalJSON() ([]byte, error) {
	type Properties AWSNeptuneDBSubnetGroup
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
func (r *AWSNeptuneDBSubnetGroup) UnmarshalJSON(b []byte) error {
	type Properties AWSNeptuneDBSubnetGroup
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
		*r = AWSNeptuneDBSubnetGroup(*res.Properties)
	}

	return nil
}

// GetAllAWSNeptuneDBSubnetGroupResources retrieves all AWSNeptuneDBSubnetGroup items from an AWS CloudFormation template
func (t *Template) GetAllAWSNeptuneDBSubnetGroupResources() map[string]AWSNeptuneDBSubnetGroup {
	results := map[string]AWSNeptuneDBSubnetGroup{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSNeptuneDBSubnetGroup:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::Neptune::DBSubnetGroup" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSNeptuneDBSubnetGroup
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

// GetAWSNeptuneDBSubnetGroupWithName retrieves all AWSNeptuneDBSubnetGroup items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSNeptuneDBSubnetGroupWithName(name string) (AWSNeptuneDBSubnetGroup, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSNeptuneDBSubnetGroup:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::Neptune::DBSubnetGroup" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSNeptuneDBSubnetGroup
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSNeptuneDBSubnetGroup{}, errors.New("resource not found")
}
