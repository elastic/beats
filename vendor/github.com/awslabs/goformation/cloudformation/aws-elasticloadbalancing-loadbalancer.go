package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSElasticLoadBalancingLoadBalancer AWS CloudFormation Resource (AWS::ElasticLoadBalancing::LoadBalancer)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-elb.html
type AWSElasticLoadBalancingLoadBalancer struct {

	// AccessLoggingPolicy AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-elb.html#cfn-ec2-elb-accessloggingpolicy
	AccessLoggingPolicy *AWSElasticLoadBalancingLoadBalancer_AccessLoggingPolicy `json:"AccessLoggingPolicy,omitempty"`

	// AppCookieStickinessPolicy AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-elb.html#cfn-ec2-elb-appcookiestickinesspolicy
	AppCookieStickinessPolicy []AWSElasticLoadBalancingLoadBalancer_AppCookieStickinessPolicy `json:"AppCookieStickinessPolicy,omitempty"`

	// AvailabilityZones AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-elb.html#cfn-ec2-elb-availabilityzones
	AvailabilityZones []string `json:"AvailabilityZones,omitempty"`

	// ConnectionDrainingPolicy AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-elb.html#cfn-ec2-elb-connectiondrainingpolicy
	ConnectionDrainingPolicy *AWSElasticLoadBalancingLoadBalancer_ConnectionDrainingPolicy `json:"ConnectionDrainingPolicy,omitempty"`

	// ConnectionSettings AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-elb.html#cfn-ec2-elb-connectionsettings
	ConnectionSettings *AWSElasticLoadBalancingLoadBalancer_ConnectionSettings `json:"ConnectionSettings,omitempty"`

	// CrossZone AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-elb.html#cfn-ec2-elb-crosszone
	CrossZone bool `json:"CrossZone,omitempty"`

	// HealthCheck AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-elb.html#cfn-ec2-elb-healthcheck
	HealthCheck *AWSElasticLoadBalancingLoadBalancer_HealthCheck `json:"HealthCheck,omitempty"`

	// Instances AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-elb.html#cfn-ec2-elb-instances
	Instances []string `json:"Instances,omitempty"`

	// LBCookieStickinessPolicy AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-elb.html#cfn-ec2-elb-lbcookiestickinesspolicy
	LBCookieStickinessPolicy []AWSElasticLoadBalancingLoadBalancer_LBCookieStickinessPolicy `json:"LBCookieStickinessPolicy,omitempty"`

	// Listeners AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-elb.html#cfn-ec2-elb-listeners
	Listeners []AWSElasticLoadBalancingLoadBalancer_Listeners `json:"Listeners,omitempty"`

	// LoadBalancerName AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-elb.html#cfn-ec2-elb-elbname
	LoadBalancerName string `json:"LoadBalancerName,omitempty"`

	// Policies AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-elb.html#cfn-ec2-elb-policies
	Policies []AWSElasticLoadBalancingLoadBalancer_Policies `json:"Policies,omitempty"`

	// Scheme AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-elb.html#cfn-ec2-elb-scheme
	Scheme string `json:"Scheme,omitempty"`

	// SecurityGroups AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-elb.html#cfn-ec2-elb-securitygroups
	SecurityGroups []string `json:"SecurityGroups,omitempty"`

	// Subnets AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-elb.html#cfn-ec2-elb-subnets
	Subnets []string `json:"Subnets,omitempty"`

	// Tags AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-elb.html#cfn-elasticloadbalancing-loadbalancer-tags
	Tags []Tag `json:"Tags,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSElasticLoadBalancingLoadBalancer) AWSCloudFormationType() string {
	return "AWS::ElasticLoadBalancing::LoadBalancer"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSElasticLoadBalancingLoadBalancer) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSElasticLoadBalancingLoadBalancer) MarshalJSON() ([]byte, error) {
	type Properties AWSElasticLoadBalancingLoadBalancer
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
func (r *AWSElasticLoadBalancingLoadBalancer) UnmarshalJSON(b []byte) error {
	type Properties AWSElasticLoadBalancingLoadBalancer
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
		*r = AWSElasticLoadBalancingLoadBalancer(*res.Properties)
	}

	return nil
}

// GetAllAWSElasticLoadBalancingLoadBalancerResources retrieves all AWSElasticLoadBalancingLoadBalancer items from an AWS CloudFormation template
func (t *Template) GetAllAWSElasticLoadBalancingLoadBalancerResources() map[string]AWSElasticLoadBalancingLoadBalancer {
	results := map[string]AWSElasticLoadBalancingLoadBalancer{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSElasticLoadBalancingLoadBalancer:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::ElasticLoadBalancing::LoadBalancer" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSElasticLoadBalancingLoadBalancer
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

// GetAWSElasticLoadBalancingLoadBalancerWithName retrieves all AWSElasticLoadBalancingLoadBalancer items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSElasticLoadBalancingLoadBalancerWithName(name string) (AWSElasticLoadBalancingLoadBalancer, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSElasticLoadBalancingLoadBalancer:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::ElasticLoadBalancing::LoadBalancer" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSElasticLoadBalancingLoadBalancer
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSElasticLoadBalancingLoadBalancer{}, errors.New("resource not found")
}
