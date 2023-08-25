package secretsmanager

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

const (
	StepCreate = "createSecret"
	StepSet    = "setSecret"
	StepTest   = "testSecret"
	StepFinish = "finishSecret"
)

var (
	ErrRotationInvalidStep = errors.New("invalid rotation step")
	ErrRotationDisabled    = errors.New("rotation disabled")
)

// Rotator interface presents a service that is able to:
//   - Create new version of a secretsmanager secret;
//   - Update downstream services/resources to use the new version;
//   - Test the newly updated version of the secret within the scope of the related services/resources
type Rotator interface {
	RotationEnabled(ctx context.Context, secretARN string) error
	Create(ctx context.Context, secretARN, token string) error
	Set(ctx context.Context, secretARN, token string, fn func(ctx context.Context, current, pending string) error) error
	Test(ctx context.Context, secretARN, token string, fn func(ctx context.Context, pending string) error) error
	Finish(ctx context.Context, secretARN, token string) error
}

const (
	VersionCurrent  = "AWSCURRENT"
	VersionPrevious = "AWSPREVIOUS"
	VersionPending  = "AWSPENDING"
)

// DefaultRotator implements Rotator
type DefaultRotator struct {
	client ClientAPI
}

var _ Rotator = &DefaultRotator{}

func NewDefaultRotator(cli ClientAPI) *DefaultRotator {
	return &DefaultRotator{
		client: cli,
	}
}

func (r *DefaultRotator) RotationEnabled(ctx context.Context, secretARN string) error {
	out, err := r.client.DescribeSecret(ctx, &secretsmanager.DescribeSecretInput{
		SecretId: aws.String(secretARN),
	})
	if err != nil {
		return err
	}
	if !aws.ToBool(out.RotationEnabled) {
		return fmt.Errorf("%w for %s", ErrRotationDisabled, secretARN)
	}
	return nil
}

// Create implements Rotator.
func (r *DefaultRotator) Create(ctx context.Context, secretARN string, token string) error {
	// Make sure secret already has a value
	_, err := r.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretARN),
		VersionStage: aws.String(VersionCurrent),
	})
	if err != nil {
		return err
	}

	// Check if secret has an ongoing rotation then abort the process
	_, err = r.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretARN),
		VersionStage: aws.String(VersionPending),
		VersionId:    aws.String(token),
	})
	if err == nil {
		return nil
	}

	// 'ResourceNotFoundException' indicates that no ongoing rotation is occurring.
	// If so, generates and sets a new secret value.
	var te *types.ResourceNotFoundException
	if !errors.As(err, &te) {
		return err
	}

	password, err := r.client.GetRandomPassword(ctx, &secretsmanager.GetRandomPasswordInput{
		ExcludePunctuation:      aws.Bool(false),
		IncludeSpace:            aws.Bool(false),
		PasswordLength:          aws.Int64(64),
		RequireEachIncludedType: aws.Bool(true),
	})
	if err != nil {
		return err
	}
	_, err = r.client.PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{
		SecretId:      aws.String(secretARN),
		VersionStages: []string{VersionPending},
		SecretString:  password.RandomPassword,
	})
	if err != nil {
		return err
	}

	return nil
}

// Finish implements Rotator.
func (r *DefaultRotator) Finish(ctx context.Context, secretARN string, token string) error {
	// Get secret associated versions
	out, err := r.client.DescribeSecret(ctx, &secretsmanager.DescribeSecretInput{
		SecretId: aws.String(secretARN),
	})
	if err != nil {
		return err
	}

	current := ""
	if versions := out.VersionIdsToStages; versions != nil {
	LOOP:
		for ver, stages := range versions {
			for _, stage := range stages {
				if stage == VersionCurrent {
					// The correct version is already marked as current, return
					if token == ver {
						return nil
					}
					current = ver
					break LOOP
				}
			}
		}
	}

	if _, err = r.client.UpdateSecretVersionStage(
		ctx, &secretsmanager.UpdateSecretVersionStageInput{
			SecretId:            aws.String(secretARN),
			VersionStage:        aws.String(VersionCurrent),
			MoveToVersionId:     aws.String(token),
			RemoveFromVersionId: aws.String(current),
		},
	); err != nil {
		return err
	}

	return nil
}

// Set implements Rotator.
func (r *DefaultRotator) Set(ctx context.Context, secretARN string, token string, fn func(ctx context.Context, current, pending string) error) error {
	if fn == nil {
		return nil
	}

	// Make sure an ongoing rotation exists, and it's the current one (based on token)
	pending, err := r.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretARN),
		VersionStage: aws.String(VersionPending),
		VersionId:    aws.String(token),
	})
	if err != nil {
		return nil
	}

	// Get the CURRENT value and pass both PENDING and CURRENT to the updater function
	current, err := r.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretARN),
		VersionStage: aws.String(VersionCurrent),
	})
	if err != nil {
		return nil
	}

	if err := fn(ctx, aws.ToString(current.SecretString), aws.ToString(pending.SecretString)); err != nil {
		return err
	}

	return nil
}

// Test implements Rotator.
func (r *DefaultRotator) Test(ctx context.Context, secretARN, token string, fn func(ctx context.Context, pending string) error) error {
	if fn == nil {
		return nil
	}

	pending, err := r.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretARN),
		VersionStage: aws.String(VersionPending),
		VersionId:    aws.String(token),
	})
	if err != nil {
		return nil
	}

	if err := fn(ctx, aws.ToString(pending.SecretString)); err != nil {
		return err
	}

	return nil
}
