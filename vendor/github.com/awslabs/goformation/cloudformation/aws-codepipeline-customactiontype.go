package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSCodePipelineCustomActionType AWS CloudFormation Resource (AWS::CodePipeline::CustomActionType)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codepipeline-customactiontype.html
type AWSCodePipelineCustomActionType struct {

	// Category AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codepipeline-customactiontype.html#cfn-codepipeline-customactiontype-category
	Category string `json:"Category,omitempty"`

	// ConfigurationProperties AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codepipeline-customactiontype.html#cfn-codepipeline-customactiontype-configurationproperties
	ConfigurationProperties []AWSCodePipelineCustomActionType_ConfigurationProperties `json:"ConfigurationProperties,omitempty"`

	// InputArtifactDetails AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codepipeline-customactiontype.html#cfn-codepipeline-customactiontype-inputartifactdetails
	InputArtifactDetails *AWSCodePipelineCustomActionType_ArtifactDetails `json:"InputArtifactDetails,omitempty"`

	// OutputArtifactDetails AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codepipeline-customactiontype.html#cfn-codepipeline-customactiontype-outputartifactdetails
	OutputArtifactDetails *AWSCodePipelineCustomActionType_ArtifactDetails `json:"OutputArtifactDetails,omitempty"`

	// Provider AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codepipeline-customactiontype.html#cfn-codepipeline-customactiontype-provider
	Provider string `json:"Provider,omitempty"`

	// Settings AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codepipeline-customactiontype.html#cfn-codepipeline-customactiontype-settings
	Settings *AWSCodePipelineCustomActionType_Settings `json:"Settings,omitempty"`

	// Version AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codepipeline-customactiontype.html#cfn-codepipeline-customactiontype-version
	Version string `json:"Version,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCodePipelineCustomActionType) AWSCloudFormationType() string {
	return "AWS::CodePipeline::CustomActionType"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCodePipelineCustomActionType) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSCodePipelineCustomActionType) MarshalJSON() ([]byte, error) {
	type Properties AWSCodePipelineCustomActionType
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
func (r *AWSCodePipelineCustomActionType) UnmarshalJSON(b []byte) error {
	type Properties AWSCodePipelineCustomActionType
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
		*r = AWSCodePipelineCustomActionType(*res.Properties)
	}

	return nil
}

// GetAllAWSCodePipelineCustomActionTypeResources retrieves all AWSCodePipelineCustomActionType items from an AWS CloudFormation template
func (t *Template) GetAllAWSCodePipelineCustomActionTypeResources() map[string]AWSCodePipelineCustomActionType {
	results := map[string]AWSCodePipelineCustomActionType{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSCodePipelineCustomActionType:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::CodePipeline::CustomActionType" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSCodePipelineCustomActionType
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

// GetAWSCodePipelineCustomActionTypeWithName retrieves all AWSCodePipelineCustomActionType items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSCodePipelineCustomActionTypeWithName(name string) (AWSCodePipelineCustomActionType, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSCodePipelineCustomActionType:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::CodePipeline::CustomActionType" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSCodePipelineCustomActionType
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSCodePipelineCustomActionType{}, errors.New("resource not found")
}
