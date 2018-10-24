package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSElasticLoadBalancingV2ListenerCertificate AWS CloudFormation Resource (AWS::ElasticLoadBalancingV2::ListenerCertificate)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-elasticloadbalancingv2-listenercertificate.html
type AWSElasticLoadBalancingV2ListenerCertificate struct {

	// Certificates AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-elasticloadbalancingv2-listenercertificate.html#cfn-elasticloadbalancingv2-listenercertificate-certificates
	Certificates []AWSElasticLoadBalancingV2ListenerCertificate_Certificate `json:"Certificates,omitempty"`

	// ListenerArn AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-elasticloadbalancingv2-listenercertificate.html#cfn-elasticloadbalancingv2-listenercertificate-listenerarn
	ListenerArn string `json:"ListenerArn,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSElasticLoadBalancingV2ListenerCertificate) AWSCloudFormationType() string {
	return "AWS::ElasticLoadBalancingV2::ListenerCertificate"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSElasticLoadBalancingV2ListenerCertificate) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSElasticLoadBalancingV2ListenerCertificate) MarshalJSON() ([]byte, error) {
	type Properties AWSElasticLoadBalancingV2ListenerCertificate
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
func (r *AWSElasticLoadBalancingV2ListenerCertificate) UnmarshalJSON(b []byte) error {
	type Properties AWSElasticLoadBalancingV2ListenerCertificate
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
		*r = AWSElasticLoadBalancingV2ListenerCertificate(*res.Properties)
	}

	return nil
}

// GetAllAWSElasticLoadBalancingV2ListenerCertificateResources retrieves all AWSElasticLoadBalancingV2ListenerCertificate items from an AWS CloudFormation template
func (t *Template) GetAllAWSElasticLoadBalancingV2ListenerCertificateResources() map[string]AWSElasticLoadBalancingV2ListenerCertificate {
	results := map[string]AWSElasticLoadBalancingV2ListenerCertificate{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSElasticLoadBalancingV2ListenerCertificate:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::ElasticLoadBalancingV2::ListenerCertificate" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSElasticLoadBalancingV2ListenerCertificate
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

// GetAWSElasticLoadBalancingV2ListenerCertificateWithName retrieves all AWSElasticLoadBalancingV2ListenerCertificate items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSElasticLoadBalancingV2ListenerCertificateWithName(name string) (AWSElasticLoadBalancingV2ListenerCertificate, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSElasticLoadBalancingV2ListenerCertificate:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::ElasticLoadBalancingV2::ListenerCertificate" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSElasticLoadBalancingV2ListenerCertificate
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSElasticLoadBalancingV2ListenerCertificate{}, errors.New("resource not found")
}
