package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSWorkSpacesWorkspace AWS CloudFormation Resource (AWS::WorkSpaces::Workspace)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-workspaces-workspace.html
type AWSWorkSpacesWorkspace struct {

	// BundleId AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-workspaces-workspace.html#cfn-workspaces-workspace-bundleid
	BundleId string `json:"BundleId,omitempty"`

	// DirectoryId AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-workspaces-workspace.html#cfn-workspaces-workspace-directoryid
	DirectoryId string `json:"DirectoryId,omitempty"`

	// RootVolumeEncryptionEnabled AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-workspaces-workspace.html#cfn-workspaces-workspace-rootvolumeencryptionenabled
	RootVolumeEncryptionEnabled bool `json:"RootVolumeEncryptionEnabled,omitempty"`

	// UserName AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-workspaces-workspace.html#cfn-workspaces-workspace-username
	UserName string `json:"UserName,omitempty"`

	// UserVolumeEncryptionEnabled AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-workspaces-workspace.html#cfn-workspaces-workspace-uservolumeencryptionenabled
	UserVolumeEncryptionEnabled bool `json:"UserVolumeEncryptionEnabled,omitempty"`

	// VolumeEncryptionKey AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-workspaces-workspace.html#cfn-workspaces-workspace-volumeencryptionkey
	VolumeEncryptionKey string `json:"VolumeEncryptionKey,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSWorkSpacesWorkspace) AWSCloudFormationType() string {
	return "AWS::WorkSpaces::Workspace"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSWorkSpacesWorkspace) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSWorkSpacesWorkspace) MarshalJSON() ([]byte, error) {
	type Properties AWSWorkSpacesWorkspace
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
func (r *AWSWorkSpacesWorkspace) UnmarshalJSON(b []byte) error {
	type Properties AWSWorkSpacesWorkspace
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
		*r = AWSWorkSpacesWorkspace(*res.Properties)
	}

	return nil
}

// GetAllAWSWorkSpacesWorkspaceResources retrieves all AWSWorkSpacesWorkspace items from an AWS CloudFormation template
func (t *Template) GetAllAWSWorkSpacesWorkspaceResources() map[string]AWSWorkSpacesWorkspace {
	results := map[string]AWSWorkSpacesWorkspace{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSWorkSpacesWorkspace:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::WorkSpaces::Workspace" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSWorkSpacesWorkspace
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

// GetAWSWorkSpacesWorkspaceWithName retrieves all AWSWorkSpacesWorkspace items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSWorkSpacesWorkspaceWithName(name string) (AWSWorkSpacesWorkspace, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSWorkSpacesWorkspace:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::WorkSpaces::Workspace" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSWorkSpacesWorkspace
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSWorkSpacesWorkspace{}, errors.New("resource not found")
}
