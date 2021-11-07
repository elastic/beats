package s3manager

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3iface"
)

// GetBucketRegion will attempt to get the region for a bucket using the
// regionHint to determine which AWS partition to perform the query on.
//
// The request will not be signed, and will not use your AWS aws.
//
// A "NotFound" error code will be returned if the bucket does not exist in
// the AWS partition the regionHint belongs to.
//
// For example to get the region of a bucket which exists in "eu-central-1"
// you could provide a region hint of "us-west-2".
//
//    cfg, err := external.LoadDefaultAWSConfig()
//
//    bucket := "my-bucket"
//    region, err := s3manager.GetBucketRegion(ctx, cfg, bucket, "us-west-2")
//    if err != nil {
//        if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "NotFound" {
//             fmt.Fprintf(os.Stderr, "unable to find bucket %s's region not found\n", bucket)
//        }
//        return err
//    }
//    fmt.Printf("Bucket %s is in %s region\n", bucket, region)
//
func GetBucketRegion(ctx context.Context, cfg aws.Config, bucket, regionHint string, opts ...aws.Option) (string, error) {
	cfg = cfg.Copy()
	cfg.Region = regionHint

	svc := s3.New(cfg)
	svc.ForcePathStyle = true
	svc.Credentials = aws.AnonymousCredentials
	return GetBucketRegionWithClient(ctx, svc, bucket, opts...)
}

const bucketRegionHeader = "X-Amz-Bucket-Region"

// GetBucketRegionWithClient is the same as GetBucketRegion with the exception
// that it takes a S3 service client instead of a Session. The regionHint is
// derived from the region the S3 service client was created in.
//
// See GetBucketRegion for more information.
func GetBucketRegionWithClient(ctx context.Context, svc s3iface.ClientAPI, bucket string, opts ...aws.Option) (string, error) {
	req := svc.HeadBucketRequest(&s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})

	var bucketRegion string
	req.Handlers.Send.PushBack(func(r *aws.Request) {
		bucketRegion = r.HTTPResponse.Header.Get(bucketRegionHeader)
		if len(bucketRegion) == 0 {
			return
		}
		r.HTTPResponse.StatusCode = 200
		r.HTTPResponse.Status = "OK"
		r.Error = nil
	})

	req.ApplyOptions(opts...)

	if _, err := req.Send(ctx); err != nil {
		return "", err
	}

	bucketRegion = string(s3.NormalizeBucketLocation(s3.BucketLocationConstraint(bucketRegion)))

	return bucketRegion, nil
}
