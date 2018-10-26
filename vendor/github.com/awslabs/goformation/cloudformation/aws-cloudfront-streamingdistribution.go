package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSCloudFrontStreamingDistribution AWS CloudFormation Resource (AWS::CloudFront::StreamingDistribution)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-cloudfront-streamingdistribution.html
type AWSCloudFrontStreamingDistribution struct {

	// StreamingDistributionConfig AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-cloudfront-streamingdistribution.html#cfn-cloudfront-streamingdistribution-streamingdistributionconfig
	StreamingDistributionConfig *AWSCloudFrontStreamingDistribution_StreamingDistributionConfig `json:"StreamingDistributionConfig,omitempty"`

	// Tags AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-cloudfront-streamingdistribution.html#cfn-cloudfront-streamingdistribution-tags
	Tags []Tag `json:"Tags,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCloudFrontStreamingDistribution) AWSCloudFormationType() string {
	return "AWS::CloudFront::StreamingDistribution"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCloudFrontStreamingDistribution) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSCloudFrontStreamingDistribution) MarshalJSON() ([]byte, error) {
	type Properties AWSCloudFrontStreamingDistribution
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
func (r *AWSCloudFrontStreamingDistribution) UnmarshalJSON(b []byte) error {
	type Properties AWSCloudFrontStreamingDistribution
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
		*r = AWSCloudFrontStreamingDistribution(*res.Properties)
	}

	return nil
}

// GetAllAWSCloudFrontStreamingDistributionResources retrieves all AWSCloudFrontStreamingDistribution items from an AWS CloudFormation template
func (t *Template) GetAllAWSCloudFrontStreamingDistributionResources() map[string]AWSCloudFrontStreamingDistribution {
	results := map[string]AWSCloudFrontStreamingDistribution{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSCloudFrontStreamingDistribution:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::CloudFront::StreamingDistribution" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSCloudFrontStreamingDistribution
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

// GetAWSCloudFrontStreamingDistributionWithName retrieves all AWSCloudFrontStreamingDistribution items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSCloudFrontStreamingDistributionWithName(name string) (AWSCloudFrontStreamingDistribution, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSCloudFrontStreamingDistribution:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::CloudFront::StreamingDistribution" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSCloudFrontStreamingDistribution
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSCloudFrontStreamingDistribution{}, errors.New("resource not found")
}
