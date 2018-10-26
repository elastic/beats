package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSSageMakerNotebookInstanceLifecycleConfig AWS CloudFormation Resource (AWS::SageMaker::NotebookInstanceLifecycleConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-sagemaker-notebookinstancelifecycleconfig.html
type AWSSageMakerNotebookInstanceLifecycleConfig struct {

	// NotebookInstanceLifecycleConfigName AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-sagemaker-notebookinstancelifecycleconfig.html#cfn-sagemaker-notebookinstancelifecycleconfig-notebookinstancelifecycleconfigname
	NotebookInstanceLifecycleConfigName string `json:"NotebookInstanceLifecycleConfigName,omitempty"`

	// OnCreate AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-sagemaker-notebookinstancelifecycleconfig.html#cfn-sagemaker-notebookinstancelifecycleconfig-oncreate
	OnCreate []AWSSageMakerNotebookInstanceLifecycleConfig_NotebookInstanceLifecycleHook `json:"OnCreate,omitempty"`

	// OnStart AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-sagemaker-notebookinstancelifecycleconfig.html#cfn-sagemaker-notebookinstancelifecycleconfig-onstart
	OnStart []AWSSageMakerNotebookInstanceLifecycleConfig_NotebookInstanceLifecycleHook `json:"OnStart,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSSageMakerNotebookInstanceLifecycleConfig) AWSCloudFormationType() string {
	return "AWS::SageMaker::NotebookInstanceLifecycleConfig"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSSageMakerNotebookInstanceLifecycleConfig) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSSageMakerNotebookInstanceLifecycleConfig) MarshalJSON() ([]byte, error) {
	type Properties AWSSageMakerNotebookInstanceLifecycleConfig
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
func (r *AWSSageMakerNotebookInstanceLifecycleConfig) UnmarshalJSON(b []byte) error {
	type Properties AWSSageMakerNotebookInstanceLifecycleConfig
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
		*r = AWSSageMakerNotebookInstanceLifecycleConfig(*res.Properties)
	}

	return nil
}

// GetAllAWSSageMakerNotebookInstanceLifecycleConfigResources retrieves all AWSSageMakerNotebookInstanceLifecycleConfig items from an AWS CloudFormation template
func (t *Template) GetAllAWSSageMakerNotebookInstanceLifecycleConfigResources() map[string]AWSSageMakerNotebookInstanceLifecycleConfig {
	results := map[string]AWSSageMakerNotebookInstanceLifecycleConfig{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSSageMakerNotebookInstanceLifecycleConfig:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::SageMaker::NotebookInstanceLifecycleConfig" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSSageMakerNotebookInstanceLifecycleConfig
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

// GetAWSSageMakerNotebookInstanceLifecycleConfigWithName retrieves all AWSSageMakerNotebookInstanceLifecycleConfig items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSSageMakerNotebookInstanceLifecycleConfigWithName(name string) (AWSSageMakerNotebookInstanceLifecycleConfig, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSSageMakerNotebookInstanceLifecycleConfig:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::SageMaker::NotebookInstanceLifecycleConfig" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSSageMakerNotebookInstanceLifecycleConfig
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSSageMakerNotebookInstanceLifecycleConfig{}, errors.New("resource not found")
}
