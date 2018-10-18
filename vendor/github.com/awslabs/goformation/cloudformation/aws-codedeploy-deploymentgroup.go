package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSCodeDeployDeploymentGroup AWS CloudFormation Resource (AWS::CodeDeploy::DeploymentGroup)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codedeploy-deploymentgroup.html
type AWSCodeDeployDeploymentGroup struct {

	// AlarmConfiguration AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codedeploy-deploymentgroup.html#cfn-codedeploy-deploymentgroup-alarmconfiguration
	AlarmConfiguration *AWSCodeDeployDeploymentGroup_AlarmConfiguration `json:"AlarmConfiguration,omitempty"`

	// ApplicationName AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codedeploy-deploymentgroup.html#cfn-codedeploy-deploymentgroup-applicationname
	ApplicationName string `json:"ApplicationName,omitempty"`

	// AutoRollbackConfiguration AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codedeploy-deploymentgroup.html#cfn-codedeploy-deploymentgroup-autorollbackconfiguration
	AutoRollbackConfiguration *AWSCodeDeployDeploymentGroup_AutoRollbackConfiguration `json:"AutoRollbackConfiguration,omitempty"`

	// AutoScalingGroups AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codedeploy-deploymentgroup.html#cfn-codedeploy-deploymentgroup-autoscalinggroups
	AutoScalingGroups []string `json:"AutoScalingGroups,omitempty"`

	// Deployment AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codedeploy-deploymentgroup.html#cfn-codedeploy-deploymentgroup-deployment
	Deployment *AWSCodeDeployDeploymentGroup_Deployment `json:"Deployment,omitempty"`

	// DeploymentConfigName AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codedeploy-deploymentgroup.html#cfn-codedeploy-deploymentgroup-deploymentconfigname
	DeploymentConfigName string `json:"DeploymentConfigName,omitempty"`

	// DeploymentGroupName AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codedeploy-deploymentgroup.html#cfn-codedeploy-deploymentgroup-deploymentgroupname
	DeploymentGroupName string `json:"DeploymentGroupName,omitempty"`

	// DeploymentStyle AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codedeploy-deploymentgroup.html#cfn-codedeploy-deploymentgroup-deploymentstyle
	DeploymentStyle *AWSCodeDeployDeploymentGroup_DeploymentStyle `json:"DeploymentStyle,omitempty"`

	// Ec2TagFilters AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codedeploy-deploymentgroup.html#cfn-codedeploy-deploymentgroup-ec2tagfilters
	Ec2TagFilters []AWSCodeDeployDeploymentGroup_EC2TagFilter `json:"Ec2TagFilters,omitempty"`

	// LoadBalancerInfo AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codedeploy-deploymentgroup.html#cfn-codedeploy-deploymentgroup-loadbalancerinfo
	LoadBalancerInfo *AWSCodeDeployDeploymentGroup_LoadBalancerInfo `json:"LoadBalancerInfo,omitempty"`

	// OnPremisesInstanceTagFilters AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codedeploy-deploymentgroup.html#cfn-codedeploy-deploymentgroup-onpremisesinstancetagfilters
	OnPremisesInstanceTagFilters []AWSCodeDeployDeploymentGroup_TagFilter `json:"OnPremisesInstanceTagFilters,omitempty"`

	// ServiceRoleArn AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codedeploy-deploymentgroup.html#cfn-codedeploy-deploymentgroup-servicerolearn
	ServiceRoleArn string `json:"ServiceRoleArn,omitempty"`

	// TriggerConfigurations AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-codedeploy-deploymentgroup.html#cfn-codedeploy-deploymentgroup-triggerconfigurations
	TriggerConfigurations []AWSCodeDeployDeploymentGroup_TriggerConfig `json:"TriggerConfigurations,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCodeDeployDeploymentGroup) AWSCloudFormationType() string {
	return "AWS::CodeDeploy::DeploymentGroup"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCodeDeployDeploymentGroup) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSCodeDeployDeploymentGroup) MarshalJSON() ([]byte, error) {
	type Properties AWSCodeDeployDeploymentGroup
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
func (r *AWSCodeDeployDeploymentGroup) UnmarshalJSON(b []byte) error {
	type Properties AWSCodeDeployDeploymentGroup
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
		*r = AWSCodeDeployDeploymentGroup(*res.Properties)
	}

	return nil
}

// GetAllAWSCodeDeployDeploymentGroupResources retrieves all AWSCodeDeployDeploymentGroup items from an AWS CloudFormation template
func (t *Template) GetAllAWSCodeDeployDeploymentGroupResources() map[string]AWSCodeDeployDeploymentGroup {
	results := map[string]AWSCodeDeployDeploymentGroup{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSCodeDeployDeploymentGroup:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::CodeDeploy::DeploymentGroup" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSCodeDeployDeploymentGroup
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

// GetAWSCodeDeployDeploymentGroupWithName retrieves all AWSCodeDeployDeploymentGroup items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSCodeDeployDeploymentGroupWithName(name string) (AWSCodeDeployDeploymentGroup, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSCodeDeployDeploymentGroup:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::CodeDeploy::DeploymentGroup" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSCodeDeployDeploymentGroup
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSCodeDeployDeploymentGroup{}, errors.New("resource not found")
}
