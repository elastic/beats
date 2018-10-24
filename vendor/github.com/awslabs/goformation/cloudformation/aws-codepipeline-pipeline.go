package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSCodePipelinePipeline AWS CloudFormation Resource (AWS::CodePipeline::Pipeline)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codepipeline-pipeline.html
type AWSCodePipelinePipeline struct {

	// ArtifactStore AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codepipeline-pipeline.html#cfn-codepipeline-pipeline-artifactstore
	ArtifactStore *AWSCodePipelinePipeline_ArtifactStore `json:"ArtifactStore,omitempty"`

	// DisableInboundStageTransitions AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codepipeline-pipeline.html#cfn-codepipeline-pipeline-disableinboundstagetransitions
	DisableInboundStageTransitions []AWSCodePipelinePipeline_StageTransition `json:"DisableInboundStageTransitions,omitempty"`

	// Name AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codepipeline-pipeline.html#cfn-codepipeline-pipeline-name
	Name string `json:"Name,omitempty"`

	// RestartExecutionOnUpdate AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codepipeline-pipeline.html#cfn-codepipeline-pipeline-restartexecutiononupdate
	RestartExecutionOnUpdate bool `json:"RestartExecutionOnUpdate,omitempty"`

	// RoleArn AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codepipeline-pipeline.html#cfn-codepipeline-pipeline-rolearn
	RoleArn string `json:"RoleArn,omitempty"`

	// Stages AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codepipeline-pipeline.html#cfn-codepipeline-pipeline-stages
	Stages []AWSCodePipelinePipeline_StageDeclaration `json:"Stages,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCodePipelinePipeline) AWSCloudFormationType() string {
	return "AWS::CodePipeline::Pipeline"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCodePipelinePipeline) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSCodePipelinePipeline) MarshalJSON() ([]byte, error) {
	type Properties AWSCodePipelinePipeline
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
func (r *AWSCodePipelinePipeline) UnmarshalJSON(b []byte) error {
	type Properties AWSCodePipelinePipeline
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
		*r = AWSCodePipelinePipeline(*res.Properties)
	}

	return nil
}

// GetAllAWSCodePipelinePipelineResources retrieves all AWSCodePipelinePipeline items from an AWS CloudFormation template
func (t *Template) GetAllAWSCodePipelinePipelineResources() map[string]AWSCodePipelinePipeline {
	results := map[string]AWSCodePipelinePipeline{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSCodePipelinePipeline:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::CodePipeline::Pipeline" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSCodePipelinePipeline
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

// GetAWSCodePipelinePipelineWithName retrieves all AWSCodePipelinePipeline items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSCodePipelinePipelineWithName(name string) (AWSCodePipelinePipeline, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSCodePipelinePipeline:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::CodePipeline::Pipeline" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSCodePipelinePipeline
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSCodePipelinePipeline{}, errors.New("resource not found")
}
