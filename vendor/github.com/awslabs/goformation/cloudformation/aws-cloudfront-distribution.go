package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSCloudFrontDistribution AWS CloudFormation Resource (AWS::CloudFront::Distribution)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-cloudfront-distribution.html
type AWSCloudFrontDistribution struct {

	// DistributionConfig AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-cloudfront-distribution.html#cfn-cloudfront-distribution-distributionconfig
	DistributionConfig *AWSCloudFrontDistribution_DistributionConfig `json:"DistributionConfig,omitempty"`

	// Tags AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-cloudfront-distribution.html#cfn-cloudfront-distribution-tags
	Tags []Tag `json:"Tags,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCloudFrontDistribution) AWSCloudFormationType() string {
	return "AWS::CloudFront::Distribution"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCloudFrontDistribution) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSCloudFrontDistribution) MarshalJSON() ([]byte, error) {
	type Properties AWSCloudFrontDistribution
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
func (r *AWSCloudFrontDistribution) UnmarshalJSON(b []byte) error {
	type Properties AWSCloudFrontDistribution
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
		*r = AWSCloudFrontDistribution(*res.Properties)
	}

	return nil
}

// GetAllAWSCloudFrontDistributionResources retrieves all AWSCloudFrontDistribution items from an AWS CloudFormation template
func (t *Template) GetAllAWSCloudFrontDistributionResources() map[string]AWSCloudFrontDistribution {
	results := map[string]AWSCloudFrontDistribution{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSCloudFrontDistribution:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::CloudFront::Distribution" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSCloudFrontDistribution
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

// GetAWSCloudFrontDistributionWithName retrieves all AWSCloudFrontDistribution items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSCloudFrontDistributionWithName(name string) (AWSCloudFrontDistribution, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSCloudFrontDistribution:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::CloudFront::Distribution" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSCloudFrontDistribution
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSCloudFrontDistribution{}, errors.New("resource not found")
}
