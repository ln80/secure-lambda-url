package secretsmanager

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type ClientAPI interface {
	GetRandomPassword(
		context.Context, *secretsmanager.GetRandomPasswordInput, ...func(*secretsmanager.Options),
	) (*secretsmanager.GetRandomPasswordOutput, error)

	GetSecretValue(
		context.Context, *secretsmanager.GetSecretValueInput, ...func(*secretsmanager.Options),
	) (*secretsmanager.GetSecretValueOutput, error)

	PutSecretValue(
		context.Context, *secretsmanager.PutSecretValueInput, ...func(*secretsmanager.Options),
	) (*secretsmanager.PutSecretValueOutput, error)

	DescribeSecret(
		context.Context, *secretsmanager.DescribeSecretInput, ...func(*secretsmanager.Options),
	) (
		*secretsmanager.DescribeSecretOutput, error,
	)

	UpdateSecretVersionStage(
		context.Context, *secretsmanager.UpdateSecretVersionStageInput,
		...func(*secretsmanager.Options),
	) (*secretsmanager.UpdateSecretVersionStageOutput, error)
}

var _ ClientAPI = &secretsmanager.Client{}

// NewClient return a secret manager client
func NewClient(cfg aws.Config, endpoint string) ClientAPI {
	svc := secretsmanager.NewFromConfig(cfg,
		func(o *secretsmanager.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		},
	)

	return svc
}
