package beater

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type EKSProvider struct {
}

func (provider EKSProvider) DescribeCluster(cfg aws.Config, ctx context.Context, clusterName string) (*eks.DescribeClusterResponse, error) {
	svc := eks.New(cfg)
	input := &eks.DescribeClusterInput{
		Name: &clusterName,
	}

	req := svc.DescribeClusterRequest(input)
	response, err := req.Send(ctx)
	if err != nil {
		logp.Err("Failed to describe cluster %s from ecr, error - %+v", clusterName, err)
		return nil, err
	}

	return response, err
}
