package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSDirectoryServiceMicrosoftAD AWS CloudFormation Resource (AWS::DirectoryService::MicrosoftAD)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-directoryservice-microsoftad.html
type AWSDirectoryServiceMicrosoftAD struct {

	// CreateAlias AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-directoryservice-microsoftad.html#cfn-directoryservice-microsoftad-createalias
	CreateAlias bool `json:"CreateAlias,omitempty"`

	// Edition AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-directoryservice-microsoftad.html#cfn-directoryservice-microsoftad-edition
	Edition string `json:"Edition,omitempty"`

	// EnableSso AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-directoryservice-microsoftad.html#cfn-directoryservice-microsoftad-enablesso
	EnableSso bool `json:"EnableSso,omitempty"`

	// Name AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-directoryservice-microsoftad.html#cfn-directoryservice-microsoftad-name
	Name string `json:"Name,omitempty"`

	// Password AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-directoryservice-microsoftad.html#cfn-directoryservice-microsoftad-password
	Password string `json:"Password,omitempty"`

	// ShortName AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-directoryservice-microsoftad.html#cfn-directoryservice-microsoftad-shortname
	ShortName string `json:"ShortName,omitempty"`

	// VpcSettings AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-directoryservice-microsoftad.html#cfn-directoryservice-microsoftad-vpcsettings
	VpcSettings *AWSDirectoryServiceMicrosoftAD_VpcSettings `json:"VpcSettings,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSDirectoryServiceMicrosoftAD) AWSCloudFormationType() string {
	return "AWS::DirectoryService::MicrosoftAD"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSDirectoryServiceMicrosoftAD) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSDirectoryServiceMicrosoftAD) MarshalJSON() ([]byte, error) {
	type Properties AWSDirectoryServiceMicrosoftAD
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
func (r *AWSDirectoryServiceMicrosoftAD) UnmarshalJSON(b []byte) error {
	type Properties AWSDirectoryServiceMicrosoftAD
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
		*r = AWSDirectoryServiceMicrosoftAD(*res.Properties)
	}

	return nil
}

// GetAllAWSDirectoryServiceMicrosoftADResources retrieves all AWSDirectoryServiceMicrosoftAD items from an AWS CloudFormation template
func (t *Template) GetAllAWSDirectoryServiceMicrosoftADResources() map[string]AWSDirectoryServiceMicrosoftAD {
	results := map[string]AWSDirectoryServiceMicrosoftAD{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSDirectoryServiceMicrosoftAD:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::DirectoryService::MicrosoftAD" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSDirectoryServiceMicrosoftAD
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

// GetAWSDirectoryServiceMicrosoftADWithName retrieves all AWSDirectoryServiceMicrosoftAD items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSDirectoryServiceMicrosoftADWithName(name string) (AWSDirectoryServiceMicrosoftAD, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSDirectoryServiceMicrosoftAD:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::DirectoryService::MicrosoftAD" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSDirectoryServiceMicrosoftAD
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSDirectoryServiceMicrosoftAD{}, errors.New("resource not found")
}
