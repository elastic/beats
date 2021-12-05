package beater

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type EKSProvider struct {
}

func (provider EKSProvider) DescribeCluster(cfg aws.Config, ctx context.Context, clusterName string) (*eks.DescribeClusterOutput, error) {
	svc := eks.NewFromConfig(cfg)
	input := &eks.DescribeClusterInput{
		Name: &clusterName,
	}

	response, err := svc.DescribeCluster(ctx, input)
	if err != nil {
		logp.Err("Failed to describe cluster %s from ecr, error - %+v", clusterName, err)
		return nil, err
	}

	return response, err
}
