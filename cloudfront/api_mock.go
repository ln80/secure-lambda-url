package cloudfront

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
)

type MockClient struct {
	GetDistributionConfigFunc func(
		ctx context.Context, params *cloudfront.GetDistributionConfigInput, optFns ...func(*cloudfront.Options),
	) (*cloudfront.GetDistributionConfigOutput, error)
	UpdateDistributionFunc func(
		ctx context.Context, params *cloudfront.UpdateDistributionInput, optFns ...func(*cloudfront.Options),
	) (*cloudfront.UpdateDistributionOutput, error)
}

var _ ClientAPI = &MockClient{}

// DescribeSecret implements ClientAPI.
func (m *MockClient) GetDistributionConfig(
	ctx context.Context, params *cloudfront.GetDistributionConfigInput, optFns ...func(*cloudfront.Options),
) (*cloudfront.GetDistributionConfigOutput, error) {
	if m.GetDistributionConfigFunc != nil {
		return m.GetDistributionConfigFunc(ctx, params, optFns...)
	}
	return nil, nil
}

// GetSecretValue implements ClientAPI.
func (m *MockClient) UpdateDistribution(
	ctx context.Context, params *cloudfront.UpdateDistributionInput, optFns ...func(*cloudfront.Options),
) (*cloudfront.UpdateDistributionOutput, error) {
	if m.UpdateDistributionFunc != nil {
		return m.UpdateDistributionFunc(ctx, params, optFns...)
	}
	return nil, nil
}
