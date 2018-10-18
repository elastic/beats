package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSKinesisAnalyticsApplicationReferenceDataSource AWS CloudFormation Resource (AWS::KinesisAnalytics::ApplicationReferenceDataSource)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-kinesisanalytics-applicationreferencedatasource.html
type AWSKinesisAnalyticsApplicationReferenceDataSource struct {

	// ApplicationName AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-kinesisanalytics-applicationreferencedatasource.html#cfn-kinesisanalytics-applicationreferencedatasource-applicationname
	ApplicationName string `json:"ApplicationName,omitempty"`

	// ReferenceDataSource AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-kinesisanalytics-applicationreferencedatasource.html#cfn-kinesisanalytics-applicationreferencedatasource-referencedatasource
	ReferenceDataSource *AWSKinesisAnalyticsApplicationReferenceDataSource_ReferenceDataSource `json:"ReferenceDataSource,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSKinesisAnalyticsApplicationReferenceDataSource) AWSCloudFormationType() string {
	return "AWS::KinesisAnalytics::ApplicationReferenceDataSource"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSKinesisAnalyticsApplicationReferenceDataSource) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSKinesisAnalyticsApplicationReferenceDataSource) MarshalJSON() ([]byte, error) {
	type Properties AWSKinesisAnalyticsApplicationReferenceDataSource
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
func (r *AWSKinesisAnalyticsApplicationReferenceDataSource) UnmarshalJSON(b []byte) error {
	type Properties AWSKinesisAnalyticsApplicationReferenceDataSource
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
		*r = AWSKinesisAnalyticsApplicationReferenceDataSource(*res.Properties)
	}

	return nil
}

// GetAllAWSKinesisAnalyticsApplicationReferenceDataSourceResources retrieves all AWSKinesisAnalyticsApplicationReferenceDataSource items from an AWS CloudFormation template
func (t *Template) GetAllAWSKinesisAnalyticsApplicationReferenceDataSourceResources() map[string]AWSKinesisAnalyticsApplicationReferenceDataSource {
	results := map[string]AWSKinesisAnalyticsApplicationReferenceDataSource{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSKinesisAnalyticsApplicationReferenceDataSource:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::KinesisAnalytics::ApplicationReferenceDataSource" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSKinesisAnalyticsApplicationReferenceDataSource
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

// GetAWSKinesisAnalyticsApplicationReferenceDataSourceWithName retrieves all AWSKinesisAnalyticsApplicationReferenceDataSource items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSKinesisAnalyticsApplicationReferenceDataSourceWithName(name string) (AWSKinesisAnalyticsApplicationReferenceDataSource, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSKinesisAnalyticsApplicationReferenceDataSource:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::KinesisAnalytics::ApplicationReferenceDataSource" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSKinesisAnalyticsApplicationReferenceDataSource
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSKinesisAnalyticsApplicationReferenceDataSource{}, errors.New("resource not found")
}
