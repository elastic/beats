package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSElasticLoadBalancingV2Listener AWS CloudFormation Resource (AWS::ElasticLoadBalancingV2::Listener)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-elasticloadbalancingv2-listener.html
type AWSElasticLoadBalancingV2Listener struct {

	// Certificates AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-elasticloadbalancingv2-listener.html#cfn-elasticloadbalancingv2-listener-certificates
	Certificates []AWSElasticLoadBalancingV2Listener_Certificate `json:"Certificates,omitempty"`

	// DefaultActions AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-elasticloadbalancingv2-listener.html#cfn-elasticloadbalancingv2-listener-defaultactions
	DefaultActions []AWSElasticLoadBalancingV2Listener_Action `json:"DefaultActions,omitempty"`

	// LoadBalancerArn AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-elasticloadbalancingv2-listener.html#cfn-elasticloadbalancingv2-listener-loadbalancerarn
	LoadBalancerArn string `json:"LoadBalancerArn,omitempty"`

	// Port AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-elasticloadbalancingv2-listener.html#cfn-elasticloadbalancingv2-listener-port
	Port int `json:"Port,omitempty"`

	// Protocol AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-elasticloadbalancingv2-listener.html#cfn-elasticloadbalancingv2-listener-protocol
	Protocol string `json:"Protocol,omitempty"`

	// SslPolicy AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-elasticloadbalancingv2-listener.html#cfn-elasticloadbalancingv2-listener-sslpolicy
	SslPolicy string `json:"SslPolicy,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSElasticLoadBalancingV2Listener) AWSCloudFormationType() string {
	return "AWS::ElasticLoadBalancingV2::Listener"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSElasticLoadBalancingV2Listener) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSElasticLoadBalancingV2Listener) MarshalJSON() ([]byte, error) {
	type Properties AWSElasticLoadBalancingV2Listener
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
func (r *AWSElasticLoadBalancingV2Listener) UnmarshalJSON(b []byte) error {
	type Properties AWSElasticLoadBalancingV2Listener
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
		*r = AWSElasticLoadBalancingV2Listener(*res.Properties)
	}

	return nil
}

// GetAllAWSElasticLoadBalancingV2ListenerResources retrieves all AWSElasticLoadBalancingV2Listener items from an AWS CloudFormation template
func (t *Template) GetAllAWSElasticLoadBalancingV2ListenerResources() map[string]AWSElasticLoadBalancingV2Listener {
	results := map[string]AWSElasticLoadBalancingV2Listener{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSElasticLoadBalancingV2Listener:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::ElasticLoadBalancingV2::Listener" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSElasticLoadBalancingV2Listener
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

// GetAWSElasticLoadBalancingV2ListenerWithName retrieves all AWSElasticLoadBalancingV2Listener items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSElasticLoadBalancingV2ListenerWithName(name string) (AWSElasticLoadBalancingV2Listener, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSElasticLoadBalancingV2Listener:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::ElasticLoadBalancingV2::Listener" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSElasticLoadBalancingV2Listener
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSElasticLoadBalancingV2Listener{}, errors.New("resource not found")
}
