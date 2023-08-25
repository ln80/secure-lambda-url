package cloudfront

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
)

type ClientAPI interface {
	GetDistributionConfig(
		ctx context.Context, params *cloudfront.GetDistributionConfigInput, optFns ...func(*cloudfront.Options),
	) (*cloudfront.GetDistributionConfigOutput, error)

	UpdateDistribution(
		ctx context.Context, params *cloudfront.UpdateDistributionInput, optFns ...func(*cloudfront.Options),
	) (*cloudfront.UpdateDistributionOutput, error)
}

var _ ClientAPI = &cloudfront.Client{}

// NewClient return a cloudfront manager client
func NewClient(cfg aws.Config) ClientAPI {
	svc := cloudfront.NewFromConfig(cfg)

	return svc
}
