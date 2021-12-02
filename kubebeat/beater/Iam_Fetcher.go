package beater

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	types2 "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/elastic/beats/v7/libbeat/logp"
	"time"
)

type IamFetcher struct {
}

func (f IamFetcher) GetIamRolePermissions(cfg aws.Config, ctx context.Context, roleName string) (interface{}, error) {

	//List attached policy
	//iam list-attached-role-policies --role-name  chime-poc-NodeInstanceRole-ZI3XYU5TCY9X
	//For each policy we will s

	results := make([]interface{}, 0)
	policiesIdentifiers, err := f.getAllRolePolicies(cfg, ctx, roleName)
	if err != nil {
		logp.Err("Failed to list role %s policies - %+v", roleName, err)
		return nil, err
	}

	svc := iam.NewFromConfig(cfg)
	for _, policyId := range policiesIdentifiers {

		input := &iam.GetRolePolicyInput{
			PolicyName: policyId.PolicyName,
			RoleName:   &roleName,
		}
		policy, err := svc.GetRolePolicy(ctx, input)
		if err != nil {
			logp.Err("Failed to get policy %s - %+v", *policyId.PolicyName, err)
			continue
		}
		results = append(results, policy)
	}

	return results, nil
}

func (f IamFetcher) getAllRolePolicies(cfg aws.Config, ctx context.Context, roleName string) ([]types2.AttachedPolicy, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	svc := iam.NewFromConfig(cfg)
	input := &iam.ListAttachedRolePoliciesInput{
		RoleName: &roleName,
	}

	allPolicies, err := svc.ListAttachedRolePolicies(ctx, input)
	if err != nil {
		logp.Err("Failed to list role %s policies - %+v", roleName, err)
		return nil, err
	}

	return allPolicies.AttachedPolicies, err
}
