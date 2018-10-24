package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSConfigAggregationAuthorization AWS CloudFormation Resource (AWS::Config::AggregationAuthorization)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-config-aggregationauthorization.html
type AWSConfigAggregationAuthorization struct {

	// AuthorizedAccountId AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-config-aggregationauthorization.html#cfn-config-aggregationauthorization-authorizedaccountid
	AuthorizedAccountId string `json:"AuthorizedAccountId,omitempty"`

	// AuthorizedAwsRegion AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-config-aggregationauthorization.html#cfn-config-aggregationauthorization-authorizedawsregion
	AuthorizedAwsRegion string `json:"AuthorizedAwsRegion,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSConfigAggregationAuthorization) AWSCloudFormationType() string {
	return "AWS::Config::AggregationAuthorization"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSConfigAggregationAuthorization) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSConfigAggregationAuthorization) MarshalJSON() ([]byte, error) {
	type Properties AWSConfigAggregationAuthorization
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
func (r *AWSConfigAggregationAuthorization) UnmarshalJSON(b []byte) error {
	type Properties AWSConfigAggregationAuthorization
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
		*r = AWSConfigAggregationAuthorization(*res.Properties)
	}

	return nil
}

// GetAllAWSConfigAggregationAuthorizationResources retrieves all AWSConfigAggregationAuthorization items from an AWS CloudFormation template
func (t *Template) GetAllAWSConfigAggregationAuthorizationResources() map[string]AWSConfigAggregationAuthorization {
	results := map[string]AWSConfigAggregationAuthorization{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSConfigAggregationAuthorization:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::Config::AggregationAuthorization" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSConfigAggregationAuthorization
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

// GetAWSConfigAggregationAuthorizationWithName retrieves all AWSConfigAggregationAuthorization items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSConfigAggregationAuthorizationWithName(name string) (AWSConfigAggregationAuthorization, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSConfigAggregationAuthorization:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::Config::AggregationAuthorization" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSConfigAggregationAuthorization
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSConfigAggregationAuthorization{}, errors.New("resource not found")
}
