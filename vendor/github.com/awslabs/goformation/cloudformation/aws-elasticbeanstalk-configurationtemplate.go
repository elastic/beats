package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSElasticBeanstalkConfigurationTemplate AWS CloudFormation Resource (AWS::ElasticBeanstalk::ConfigurationTemplate)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-elasticbeanstalk-configurationtemplate.html
type AWSElasticBeanstalkConfigurationTemplate struct {

	// ApplicationName AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-elasticbeanstalk-configurationtemplate.html#cfn-elasticbeanstalk-configurationtemplate-applicationname
	ApplicationName string `json:"ApplicationName,omitempty"`

	// Description AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-elasticbeanstalk-configurationtemplate.html#cfn-elasticbeanstalk-configurationtemplate-description
	Description string `json:"Description,omitempty"`

	// EnvironmentId AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-elasticbeanstalk-configurationtemplate.html#cfn-elasticbeanstalk-configurationtemplate-environmentid
	EnvironmentId string `json:"EnvironmentId,omitempty"`

	// OptionSettings AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-elasticbeanstalk-configurationtemplate.html#cfn-elasticbeanstalk-configurationtemplate-optionsettings
	OptionSettings []AWSElasticBeanstalkConfigurationTemplate_ConfigurationOptionSetting `json:"OptionSettings,omitempty"`

	// PlatformArn AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-elasticbeanstalk-configurationtemplate.html#cfn-elasticbeanstalk-configurationtemplate-platformarn
	PlatformArn string `json:"PlatformArn,omitempty"`

	// SolutionStackName AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-elasticbeanstalk-configurationtemplate.html#cfn-elasticbeanstalk-configurationtemplate-solutionstackname
	SolutionStackName string `json:"SolutionStackName,omitempty"`

	// SourceConfiguration AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-elasticbeanstalk-configurationtemplate.html#cfn-elasticbeanstalk-configurationtemplate-sourceconfiguration
	SourceConfiguration *AWSElasticBeanstalkConfigurationTemplate_SourceConfiguration `json:"SourceConfiguration,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSElasticBeanstalkConfigurationTemplate) AWSCloudFormationType() string {
	return "AWS::ElasticBeanstalk::ConfigurationTemplate"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSElasticBeanstalkConfigurationTemplate) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSElasticBeanstalkConfigurationTemplate) MarshalJSON() ([]byte, error) {
	type Properties AWSElasticBeanstalkConfigurationTemplate
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
func (r *AWSElasticBeanstalkConfigurationTemplate) UnmarshalJSON(b []byte) error {
	type Properties AWSElasticBeanstalkConfigurationTemplate
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
		*r = AWSElasticBeanstalkConfigurationTemplate(*res.Properties)
	}

	return nil
}

// GetAllAWSElasticBeanstalkConfigurationTemplateResources retrieves all AWSElasticBeanstalkConfigurationTemplate items from an AWS CloudFormation template
func (t *Template) GetAllAWSElasticBeanstalkConfigurationTemplateResources() map[string]AWSElasticBeanstalkConfigurationTemplate {
	results := map[string]AWSElasticBeanstalkConfigurationTemplate{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSElasticBeanstalkConfigurationTemplate:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::ElasticBeanstalk::ConfigurationTemplate" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSElasticBeanstalkConfigurationTemplate
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

// GetAWSElasticBeanstalkConfigurationTemplateWithName retrieves all AWSElasticBeanstalkConfigurationTemplate items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSElasticBeanstalkConfigurationTemplateWithName(name string) (AWSElasticBeanstalkConfigurationTemplate, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSElasticBeanstalkConfigurationTemplate:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::ElasticBeanstalk::ConfigurationTemplate" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSElasticBeanstalkConfigurationTemplate
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSElasticBeanstalkConfigurationTemplate{}, errors.New("resource not found")
}
