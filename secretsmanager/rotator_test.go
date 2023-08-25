package secretsmanager

import (
	"context"
	"errors"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

func TestRotator_Create(t *testing.T) {
	ctx := context.Background()
	secret := "arn:aws:secretsmanager:eu-west-1:19cx3122:secret/fake"
	token := "arn:aws:secretsmanager:eu-west-1:19cx3122:token/fake"

	t.Run("secret must already have a current value to rotate", func(t *testing.T) {
		mockErr := &types.ResourceNotFoundException{}
		cli := &MockClient{
			GetSecretValueFunc: func(ctx context.Context, gsvi *secretsmanager.GetSecretValueInput, f ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				return nil, mockErr
			},
		}

		err := NewDefaultRotator(cli).Create(ctx, secret, token)
		if got, want := err, mockErr; !errors.Is(got, want) {
			t.Fatalf("expect %v is %v", got, want)
		}
	})

	t.Run("do not create new secret during ongoing rotation", func(t *testing.T) {
		spyCalls := int32(0)
		cli := &MockClient{
			GetSecretValueFunc: func(ctx context.Context, gsvi *secretsmanager.GetSecretValueInput, f ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				// for clarity reasons, explicitly return a value for the 'PENDING' case
				// even if it's the same as default
				if aws.ToString(gsvi.VersionStage) == VersionPending {
					return &secretsmanager.GetSecretValueOutput{}, nil
				}
				return &secretsmanager.GetSecretValueOutput{}, nil
			},

			GetRandomPasswordFunc: func(ctx context.Context, grpi *secretsmanager.GetRandomPasswordInput, f ...func(*secretsmanager.Options)) (*secretsmanager.GetRandomPasswordOutput, error) {
				atomic.AddInt32(&spyCalls, 1)
				return &secretsmanager.GetRandomPasswordOutput{}, nil
			},
			PutSecretValueFunc: func(ctx context.Context, psvi *secretsmanager.PutSecretValueInput, f ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error) {
				atomic.AddInt32(&spyCalls, 1)
				return &secretsmanager.PutSecretValueOutput{}, nil
			},
		}

		err := NewDefaultRotator(cli).Create(ctx, secret, token)
		if err != nil {
			t.Fatalf("expect err be nil, got %v", err)
		}
		if spyCalls != 0 {
			t.Fatal("expect 'put secret logic' to not be reached")
		}
	})

	t.Run("new secret should be created", func(t *testing.T) {
		mockErr := &types.ResourceNotFoundException{}
		spyCalls := int32(0)
		cli := &MockClient{
			GetSecretValueFunc: func(ctx context.Context, gsvi *secretsmanager.GetSecretValueInput, f ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				if aws.ToString(gsvi.VersionStage) == VersionPending {
					return nil, mockErr
				}
				return &secretsmanager.GetSecretValueOutput{}, nil
			},
			GetRandomPasswordFunc: func(ctx context.Context, grpi *secretsmanager.GetRandomPasswordInput, f ...func(*secretsmanager.Options)) (*secretsmanager.GetRandomPasswordOutput, error) {
				atomic.AddInt32(&spyCalls, 1)
				return &secretsmanager.GetRandomPasswordOutput{}, nil
			},
			PutSecretValueFunc: func(ctx context.Context, psvi *secretsmanager.PutSecretValueInput, f ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error) {
				atomic.AddInt32(&spyCalls, 1)
				return &secretsmanager.PutSecretValueOutput{}, nil
			},
		}

		err := NewDefaultRotator(cli).Create(ctx, secret, token)
		if err != nil {
			t.Fatalf("expect err be nil, got %v", err)
		}
		if spyCalls != 2 {
			t.Fatal("expect 'put secret logic' to be executed")
		}
	})

	t.Run("secret rotation create failed due to infra error", func(t *testing.T) {
		mockErr := errors.New("infra error")
		cli := &MockClient{
			GetSecretValueFunc: func(ctx context.Context, gsvi *secretsmanager.GetSecretValueInput, f ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				if aws.ToString(gsvi.VersionStage) == VersionPending {
					return nil, &types.ResourceNotFoundException{}
				}
				return &secretsmanager.GetSecretValueOutput{}, nil
			},
			GetRandomPasswordFunc: func(ctx context.Context, grpi *secretsmanager.GetRandomPasswordInput, f ...func(*secretsmanager.Options)) (*secretsmanager.GetRandomPasswordOutput, error) {
				return nil, mockErr
			},
		}

		err := NewDefaultRotator(cli).Create(ctx, secret, token)
		if got, want := err, mockErr; !errors.Is(got, want) {
			t.Fatalf("expect %v is %v", got, want)
		}
	})
}

func TestRotator_Set(t *testing.T) {
	ctx := context.Background()
	secret := "arn:aws:secretmanager:eu-west-1:19cx3122:secret/fake"
	token := "arn:aws:secretmanager:eu-west-1:19cx3122:token/fake"

	t.Run("ongoing rotation is required", func(t *testing.T) {
		mockErr := &types.ResourceNotFoundException{}
		spyCalls := int32(0)
		cli := &MockClient{
			GetSecretValueFunc: func(ctx context.Context, gsvi *secretsmanager.GetSecretValueInput, f ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				if aws.ToString(gsvi.VersionStage) == VersionPending {
					return nil, mockErr
				}
				return &secretsmanager.GetSecretValueOutput{}, nil
			},
		}
		fn := func(ctx context.Context, current, pending string) error {
			atomic.AddInt32(&spyCalls, 1)
			return nil
		}

		err := NewDefaultRotator(cli).Set(ctx, secret, token, fn)
		if err != nil {
			t.Fatalf("expect err be nil, got %v", err)
		}
		if spyCalls != 0 {
			t.Fatal("expect 'fn' to not be executed")
		}
	})

	t.Run("set new secret", func(t *testing.T) {
		spyCalls := int32(0)
		cli := &MockClient{
			GetSecretValueFunc: func(ctx context.Context, gsvi *secretsmanager.GetSecretValueInput, f ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				return &secretsmanager.GetSecretValueOutput{}, nil
			},
		}
		fn := func(ctx context.Context, current, pending string) error {
			atomic.AddInt32(&spyCalls, 1)
			return nil
		}

		err := NewDefaultRotator(cli).Set(ctx, secret, token, fn)
		if err != nil {
			t.Fatalf("expect err be nil, got %v", err)
		}
		if spyCalls != 1 {
			t.Fatal("expect 'fn' to be executed")
		}
	})
}

func TestRotator_Test(t *testing.T) {
	ctx := context.Background()
	secret := "arn:aws:secretmanager:eu-west-1:19cx3122:secret/fake"
	token := "arn:aws:secretmanager:eu-west-1:19cx3122:token/fake"

	t.Run("with ongoing rotation is required", func(t *testing.T) {
		mockErr := &types.ResourceNotFoundException{}
		spyCalls := int32(0)
		cli := &MockClient{
			GetSecretValueFunc: func(ctx context.Context, gsvi *secretsmanager.GetSecretValueInput, f ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				if aws.ToString(gsvi.VersionStage) == VersionPending {
					return nil, mockErr
				}
				return &secretsmanager.GetSecretValueOutput{}, nil
			},
		}
		fn := func(ctx context.Context, pending string) error {
			atomic.AddInt32(&spyCalls, 1)
			return nil
		}

		err := NewDefaultRotator(cli).Test(ctx, secret, token, fn)
		if err != nil {
			t.Fatalf("expect err be nil, got %v", err)
		}
		if spyCalls != 0 {
			t.Fatal("expect 'fn' to not be executed")
		}
	})

	t.Run("with new secret", func(t *testing.T) {
		spyCalls := int32(0)
		cli := &MockClient{
			GetSecretValueFunc: func(ctx context.Context, gsvi *secretsmanager.GetSecretValueInput, f ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				return &secretsmanager.GetSecretValueOutput{}, nil
			},
		}
		fn := func(ctx context.Context, pending string) error {
			atomic.AddInt32(&spyCalls, 1)
			return nil
		}

		err := NewDefaultRotator(cli).Test(ctx, secret, token, fn)
		if err != nil {
			t.Fatalf("expect err be nil, got %v", err)
		}
		if spyCalls != 1 {
			t.Fatal("expect 'fn' to be executed")
		}
	})

}

func TestRotator_Finish(t *testing.T) {
	ctx := context.Background()
	secret := "arn:aws:secretmanager:eu-west-1:19cx3122:secret/fake"
	token := "arn:aws:secretmanager:eu-west-1:19cx3122:token/fake"

	t.Run("with already marked as current", func(t *testing.T) {
		spyCalls := int32(0)
		cli := &MockClient{
			DescribeSecretFunc: func(ctx context.Context, dsi *secretsmanager.DescribeSecretInput, f ...func(*secretsmanager.Options)) (*secretsmanager.DescribeSecretOutput, error) {
				return &secretsmanager.DescribeSecretOutput{
					VersionIdsToStages: map[string][]string{
						token: {VersionCurrent},
					},
				}, nil
			},
			UpdateSecretVersionStageFunc: func(ctx context.Context, usvsi *secretsmanager.UpdateSecretVersionStageInput, f ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretVersionStageOutput, error) {
				atomic.AddInt32(&spyCalls, 1)
				return &secretsmanager.UpdateSecretVersionStageOutput{}, nil
			},
		}

		err := NewDefaultRotator(cli).Finish(ctx, secret, token)
		if err != nil {
			t.Fatalf("expect err be nil, got %v", err)
		}
		if spyCalls != 0 {
			t.Fatal("expect 'UpdateSecretVersionStageFunc' to not be executed")
		}
	})

	t.Run("with new version marked as current", func(t *testing.T) {
		spyCalls := int32(0)
		cli := &MockClient{
			DescribeSecretFunc: func(ctx context.Context, dsi *secretsmanager.DescribeSecretInput, f ...func(*secretsmanager.Options)) (*secretsmanager.DescribeSecretOutput, error) {
				return &secretsmanager.DescribeSecretOutput{
					VersionIdsToStages: map[string][]string{
						"new_ver": {VersionCurrent},
					},
				}, nil
			},
			UpdateSecretVersionStageFunc: func(ctx context.Context, usvsi *secretsmanager.UpdateSecretVersionStageInput, f ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretVersionStageOutput, error) {
				atomic.AddInt32(&spyCalls, 1)
				return &secretsmanager.UpdateSecretVersionStageOutput{}, nil
			},
		}

		err := NewDefaultRotator(cli).Finish(ctx, secret, token)
		if err != nil {
			t.Fatalf("expect err be nil, got %v", err)
		}
		if spyCalls != 1 {
			t.Fatal("expect 'UpdateSecretVersionStageFunc' to be executed")
		}
	})
}

func TestRotator_RotationEnabled(t *testing.T) {
	ctx := context.Background()
	secret := "arn:aws:secretmanager:eu-west-1:19cx3122:secret/fake"

	for i, tc := range []bool{true, false} {

		t.Run("tc:"+strconv.Itoa(i+1), func(t *testing.T) {
			cli := &MockClient{
				DescribeSecretFunc: func(ctx context.Context, dsi *secretsmanager.DescribeSecretInput, f ...func(*secretsmanager.Options)) (*secretsmanager.DescribeSecretOutput, error) {
					return &secretsmanager.DescribeSecretOutput{
						RotationEnabled: aws.Bool(tc),
					}, nil
				},
				UpdateSecretVersionStageFunc: func(ctx context.Context, usvsi *secretsmanager.UpdateSecretVersionStageInput, f ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretVersionStageOutput, error) {
					return &secretsmanager.UpdateSecretVersionStageOutput{}, nil
				},
			}

			err := NewDefaultRotator(cli).RotationEnabled(ctx, secret)
			if tc {
				if err != nil {
					t.Fatalf("expect err be nil, got %v", err)
				}
			} else {
				if got, want := err, ErrRotationDisabled; !errors.Is(got, want) {
					t.Fatalf("expect %v is %v", got, want)
				}
			}
		})
	}
}
