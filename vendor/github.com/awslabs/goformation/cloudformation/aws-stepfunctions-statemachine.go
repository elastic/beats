package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSStepFunctionsStateMachine AWS CloudFormation Resource (AWS::StepFunctions::StateMachine)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-stepfunctions-statemachine.html
type AWSStepFunctionsStateMachine struct {

	// DefinitionString AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-stepfunctions-statemachine.html#cfn-stepfunctions-statemachine-definitionstring
	DefinitionString string `json:"DefinitionString,omitempty"`

	// RoleArn AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-stepfunctions-statemachine.html#cfn-stepfunctions-statemachine-rolearn
	RoleArn string `json:"RoleArn,omitempty"`

	// StateMachineName AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-stepfunctions-statemachine.html#cfn-stepfunctions-statemachine-statemachinename
	StateMachineName string `json:"StateMachineName,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSStepFunctionsStateMachine) AWSCloudFormationType() string {
	return "AWS::StepFunctions::StateMachine"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSStepFunctionsStateMachine) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSStepFunctionsStateMachine) MarshalJSON() ([]byte, error) {
	type Properties AWSStepFunctionsStateMachine
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
func (r *AWSStepFunctionsStateMachine) UnmarshalJSON(b []byte) error {
	type Properties AWSStepFunctionsStateMachine
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
		*r = AWSStepFunctionsStateMachine(*res.Properties)
	}

	return nil
}

// GetAllAWSStepFunctionsStateMachineResources retrieves all AWSStepFunctionsStateMachine items from an AWS CloudFormation template
func (t *Template) GetAllAWSStepFunctionsStateMachineResources() map[string]AWSStepFunctionsStateMachine {
	results := map[string]AWSStepFunctionsStateMachine{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSStepFunctionsStateMachine:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::StepFunctions::StateMachine" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSStepFunctionsStateMachine
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

// GetAWSStepFunctionsStateMachineWithName retrieves all AWSStepFunctionsStateMachine items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSStepFunctionsStateMachineWithName(name string) (AWSStepFunctionsStateMachine, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSStepFunctionsStateMachine:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::StepFunctions::StateMachine" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSStepFunctionsStateMachine
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSStepFunctionsStateMachine{}, errors.New("resource not found")
}
