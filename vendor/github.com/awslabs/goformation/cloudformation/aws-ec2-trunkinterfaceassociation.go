package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSEC2TrunkInterfaceAssociation AWS CloudFormation Resource (AWS::EC2::TrunkInterfaceAssociation)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ec2-trunkinterfaceassociation.html
type AWSEC2TrunkInterfaceAssociation struct {

	// BranchInterfaceId AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ec2-trunkinterfaceassociation.html#cfn-ec2-trunkinterfaceassociation-branchinterfaceid
	BranchInterfaceId string `json:"BranchInterfaceId,omitempty"`

	// GREKey AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ec2-trunkinterfaceassociation.html#cfn-ec2-trunkinterfaceassociation-grekey
	GREKey int `json:"GREKey,omitempty"`

	// TrunkInterfaceId AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ec2-trunkinterfaceassociation.html#cfn-ec2-trunkinterfaceassociation-trunkinterfaceid
	TrunkInterfaceId string `json:"TrunkInterfaceId,omitempty"`

	// VLANId AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ec2-trunkinterfaceassociation.html#cfn-ec2-trunkinterfaceassociation-vlanid
	VLANId int `json:"VLANId,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEC2TrunkInterfaceAssociation) AWSCloudFormationType() string {
	return "AWS::EC2::TrunkInterfaceAssociation"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEC2TrunkInterfaceAssociation) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSEC2TrunkInterfaceAssociation) MarshalJSON() ([]byte, error) {
	type Properties AWSEC2TrunkInterfaceAssociation
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
func (r *AWSEC2TrunkInterfaceAssociation) UnmarshalJSON(b []byte) error {
	type Properties AWSEC2TrunkInterfaceAssociation
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
		*r = AWSEC2TrunkInterfaceAssociation(*res.Properties)
	}

	return nil
}

// GetAllAWSEC2TrunkInterfaceAssociationResources retrieves all AWSEC2TrunkInterfaceAssociation items from an AWS CloudFormation template
func (t *Template) GetAllAWSEC2TrunkInterfaceAssociationResources() map[string]AWSEC2TrunkInterfaceAssociation {
	results := map[string]AWSEC2TrunkInterfaceAssociation{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSEC2TrunkInterfaceAssociation:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::EC2::TrunkInterfaceAssociation" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSEC2TrunkInterfaceAssociation
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

// GetAWSEC2TrunkInterfaceAssociationWithName retrieves all AWSEC2TrunkInterfaceAssociation items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSEC2TrunkInterfaceAssociationWithName(name string) (AWSEC2TrunkInterfaceAssociation, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSEC2TrunkInterfaceAssociation:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::EC2::TrunkInterfaceAssociation" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSEC2TrunkInterfaceAssociation
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSEC2TrunkInterfaceAssociation{}, errors.New("resource not found")
}
