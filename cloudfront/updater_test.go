package cloudfront

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
)

func TestUpdater(t *testing.T) {
	ctx := context.Background()
	distID := "fake_dist"

	t.Run("with empty update funcs", func(t *testing.T) {
		cli := &MockClient{}
		u := NewDefaultUpdater(cli)

		err := u.Update(ctx, distID)
		if got, want := err, ErrUpdateFunctionIsmissing; !errors.Is(got, want) {
			t.Fatalf("expect err %v is %v", got, want)
		}
	})

	t.Run("with an update func", func(t *testing.T) {
		spyCalls := int32(0)

		cli := &MockClient{
			GetDistributionConfigFunc: func(ctx context.Context, params *cloudfront.GetDistributionConfigInput, optFns ...func(*cloudfront.Options)) (*cloudfront.GetDistributionConfigOutput, error) {
				return &cloudfront.GetDistributionConfigOutput{}, nil
			},
			UpdateDistributionFunc: func(ctx context.Context, params *cloudfront.UpdateDistributionInput, optFns ...func(*cloudfront.Options)) (*cloudfront.UpdateDistributionOutput, error) {
				return &cloudfront.UpdateDistributionOutput{}, nil
			},
		}

		u := NewDefaultUpdater(cli)

		fn := func(*DistributionConfig) {
			atomic.AddInt32(&spyCalls, 1)
		}
		err := u.Update(ctx, distID, fn)
		if err != nil {
			t.Fatalf("expect err be nil, got %v", err)
		}
		if spyCalls != 1 {
			t.Fatalf("expect 'spyCalls' be called once")
		}
	})

	t.Run("with update custom header func", func(t *testing.T) {
		spyCalls := int32(0)
		spyUpdates := int32(0)

		headerName, headerValue := "X-Custom-H", "value"

		cli := &MockClient{
			GetDistributionConfigFunc: func(ctx context.Context, params *cloudfront.GetDistributionConfigInput, optFns ...func(*cloudfront.Options)) (*cloudfront.GetDistributionConfigOutput, error) {
				atomic.AddInt32(&spyCalls, 1)
				return &cloudfront.GetDistributionConfigOutput{
					DistributionConfig: &types.DistributionConfig{
						Origins: &types.Origins{
							Items: []types.Origin{
								{
									DomainName: aws.String("domain1.xyz"),
									CustomHeaders: &types.CustomHeaders{
										Items: []types.OriginCustomHeader{
											{
												HeaderName:  &headerName,
												HeaderValue: &headerValue,
											},
										},
									},
								},
								{
									DomainName: aws.String("domain2.xyz"),
								},
							},
						},
					},
				}, nil
			},
			UpdateDistributionFunc: func(ctx context.Context, params *cloudfront.UpdateDistributionInput, optFns ...func(*cloudfront.Options)) (*cloudfront.UpdateDistributionOutput, error) {
				atomic.AddInt32(&spyCalls, 1)
				for _, origin := range params.DistributionConfig.Origins.Items {
					if origin.CustomHeaders == nil || len(origin.CustomHeaders.Items) == 0 {
						continue
					}
					for _, h := range origin.CustomHeaders.Items {
						if aws.ToString(h.HeaderName) == headerName && aws.ToString(h.HeaderValue) == headerValue {
							atomic.AddInt32(&spyUpdates, 1)
						}
					}
				}
				return &cloudfront.UpdateDistributionOutput{}, nil
			},
		}

		u := NewDefaultUpdater(cli)

		err := u.Update(ctx, distID, UpdateCustomHeaderFn(headerName, headerValue))
		if err != nil {
			t.Fatalf("expect err be nil, got %v", err)
		}
		if spyCalls != 2 {
			t.Fatalf("expect 'spyCalls' be called twice")
		}

		if spyUpdates != 1 {
			t.Fatalf("expect 'spyUpdates' be called once")
		}
	})
}
