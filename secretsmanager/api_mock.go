package secretsmanager

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type MockClient struct {
	GetRandomPasswordFunc        func(context.Context, *secretsmanager.GetRandomPasswordInput, ...func(*secretsmanager.Options)) (*secretsmanager.GetRandomPasswordOutput, error)
	GetSecretValueFunc           func(context.Context, *secretsmanager.GetSecretValueInput, ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
	PutSecretValueFunc           func(context.Context, *secretsmanager.PutSecretValueInput, ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error)
	DescribeSecretFunc           func(context.Context, *secretsmanager.DescribeSecretInput, ...func(*secretsmanager.Options)) (*secretsmanager.DescribeSecretOutput, error)
	UpdateSecretVersionStageFunc func(context.Context, *secretsmanager.UpdateSecretVersionStageInput, ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretVersionStageOutput, error)
}

var _ ClientAPI = &MockClient{}

// DescribeSecret implements ClientAPI.
func (m *MockClient) DescribeSecret(ctx context.Context, input *secretsmanager.DescribeSecretInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.DescribeSecretOutput, error) {
	if m.DescribeSecretFunc != nil {
		return m.DescribeSecretFunc(ctx, input, opts...)
	}
	return nil, nil
}

// GetSecretValue implements ClientAPI.
func (m *MockClient) GetSecretValue(ctx context.Context, input *secretsmanager.GetSecretValueInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	if m.GetSecretValueFunc != nil {
		return m.GetSecretValueFunc(ctx, input, opts...)
	}
	return nil, nil
}

// PutSecretValue implements ClientAPI.
func (m *MockClient) PutSecretValue(ctx context.Context, input *secretsmanager.PutSecretValueInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error) {
	if m.PutSecretValueFunc != nil {
		return m.PutSecretValueFunc(ctx, input, opts...)
	}
	return nil, nil
}

// UpdateSecretVersionStage implements ClientAPI.
func (m *MockClient) UpdateSecretVersionStage(ctx context.Context, input *secretsmanager.UpdateSecretVersionStageInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretVersionStageOutput, error) {
	if m.UpdateSecretVersionStageFunc != nil {
		return m.UpdateSecretVersionStageFunc(ctx, input, opts...)
	}
	return nil, nil
}

func (m *MockClient) GetRandomPassword(ctx context.Context, input *secretsmanager.GetRandomPasswordInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.GetRandomPasswordOutput, error) {
	if m.GetRandomPasswordFunc != nil {
		return m.GetRandomPasswordFunc(ctx, input, opts...)
	}
	return nil, nil
}
