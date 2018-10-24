package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSRoute53HealthCheck AWS CloudFormation Resource (AWS::Route53::HealthCheck)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-route53-healthcheck.html
type AWSRoute53HealthCheck struct {

	// HealthCheckConfig AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-route53-healthcheck.html#cfn-route53-healthcheck-healthcheckconfig
	HealthCheckConfig *AWSRoute53HealthCheck_HealthCheckConfig `json:"HealthCheckConfig,omitempty"`

	// HealthCheckTags AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-route53-healthcheck.html#cfn-route53-healthcheck-healthchecktags
	HealthCheckTags []AWSRoute53HealthCheck_HealthCheckTag `json:"HealthCheckTags,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSRoute53HealthCheck) AWSCloudFormationType() string {
	return "AWS::Route53::HealthCheck"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSRoute53HealthCheck) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSRoute53HealthCheck) MarshalJSON() ([]byte, error) {
	type Properties AWSRoute53HealthCheck
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
func (r *AWSRoute53HealthCheck) UnmarshalJSON(b []byte) error {
	type Properties AWSRoute53HealthCheck
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
		*r = AWSRoute53HealthCheck(*res.Properties)
	}

	return nil
}

// GetAllAWSRoute53HealthCheckResources retrieves all AWSRoute53HealthCheck items from an AWS CloudFormation template
func (t *Template) GetAllAWSRoute53HealthCheckResources() map[string]AWSRoute53HealthCheck {
	results := map[string]AWSRoute53HealthCheck{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSRoute53HealthCheck:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::Route53::HealthCheck" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSRoute53HealthCheck
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

// GetAWSRoute53HealthCheckWithName retrieves all AWSRoute53HealthCheck items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSRoute53HealthCheckWithName(name string) (AWSRoute53HealthCheck, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSRoute53HealthCheck:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::Route53::HealthCheck" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSRoute53HealthCheck
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSRoute53HealthCheck{}, errors.New("resource not found")
}
