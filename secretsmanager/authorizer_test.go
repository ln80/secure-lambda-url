package secretsmanager

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

func TestAuthorizer(t *testing.T) {
	ctx := context.Background()
	secret := "arn:aws:secretsmanager:eu-west-1:19cx3122:secret/fake"

	// shorter value is used for testing purposes
	ttl := 100 * time.Millisecond
	j := NewJanitor(ttl)
	j.Run(ctx, func() {})

	t.Run("with empty value", func(t *testing.T) {
		cli := &MockClient{}

		auth := NewAuthorizer(cli, j, func(ac *AuthorizerConfig) {
			ac.CoolDownPeriod = time.Second
			ac.GracePeriod = time.Second
		})

		err, _ := auth.Authorize(ctx, secret, "")
		if want, got := ErrInvalidSecretValue, err; !errors.Is(got, want) {
			t.Fatalf("expect %v, %v be equals", want, got)
		}
	})

	t.Run("with unexpected failure", func(t *testing.T) {
		spyCalls := int32(0)
		value := "a_value"

		mockErr := errors.New("infra error")
		cli := &MockClient{
			GetSecretValueFunc: func(ctx context.Context, gsvi *secretsmanager.GetSecretValueInput, f ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				atomic.AddInt32(&spyCalls, 1)
				return &secretsmanager.GetSecretValueOutput{}, mockErr
			},
		}

		auth := NewAuthorizer(cli, j, func(ac *AuthorizerConfig) {
			ac.CoolDownPeriod = time.Second
			ac.GracePeriod = time.Second
		})
		err, _ := auth.Authorize(ctx, secret, value)
		if want, got := ErrAuthorizationFailed, err; !errors.Is(got, want) {
			t.Fatalf("expect %v, %v be equals", want, got)
		}
		if !strings.Contains(err.Error(), mockErr.Error()) {
			t.Fatalf("expect err %v, contains %v", err, mockErr)
		}

		// Make sure value is not black listed, this implies a second secret API call
		err, _ = auth.Authorize(ctx, secret, value)
		if want, got := ErrAuthorizationFailed, err; !errors.Is(got, want) {
			t.Fatalf("expect %v, %v be equals", want, got)
		}
		if spyCalls != 2 {
			t.Fatal("expect 'GetSecretValue' is called twice")
		}
	})

	t.Run("with unauthorized value", func(t *testing.T) {
		spyCalls := int32(0)
		value := "invalid_value"

		cli := &MockClient{
			GetSecretValueFunc: func(ctx context.Context, gsvi *secretsmanager.GetSecretValueInput, f ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				atomic.AddInt32(&spyCalls, 1)
				return &secretsmanager.GetSecretValueOutput{}, &types.ResourceNotFoundException{}
			},
		}
		auth := NewAuthorizer(cli, j, func(ac *AuthorizerConfig) {
			ac.CoolDownPeriod = time.Second
			ac.GracePeriod = time.Second
		})

		err, _ := auth.Authorize(ctx, secret, value)
		if want, got := ErrUnauthorized, err; !errors.Is(got, want) {
			t.Fatalf("expect %v, %v be equals", want, got)
		}

		// a second 'Authorize' call should not trigger secret API call,
		// the invalid value must be already in the blacklist cache
		err, remoteCalled := auth.Authorize(ctx, secret, value)
		if want, got := ErrUnauthorized, err; !errors.Is(got, want) {
			t.Fatalf("expect %v, %v be equals", want, got)
		}
		if spyCalls != 1 {
			t.Fatal("expect 'GetSecretValue' is called once")
		}
		if !remoteCalled {
			t.Fatal("expect 'remoteCalled' be true, got false")
		}

		// wait until the cache is expired
		time.Sleep(ttl + 100*time.Millisecond)

		// a second secret API call has to be made
		err, remoteCalled = auth.Authorize(ctx, secret, value)
		if want, got := ErrUnauthorized, err; !errors.Is(got, want) {
			t.Fatalf("expect %v, %v be equals", want, got)
		}
		if spyCalls != 2 {
			t.Fatal("expect 'GetSecretValue' is called twice")
		}
		if remoteCalled {
			t.Fatal("expect 'remoteCalled' be false, got true")
		}
	})

	t.Run("with authorized value", func(t *testing.T) {
		spyCalls := int32(0)
		value := "valid_value"

		cli := &MockClient{
			GetSecretValueFunc: func(ctx context.Context, gsvi *secretsmanager.GetSecretValueInput, f ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				atomic.AddInt32(&spyCalls, 1)
				return &secretsmanager.GetSecretValueOutput{
					SecretString: aws.String(value),
				}, nil
			},
		}
		auth := NewAuthorizer(cli, j, func(ac *AuthorizerConfig) {
			ac.CoolDownPeriod = time.Second
			ac.GracePeriod = time.Second
		})

		err, _ := auth.Authorize(ctx, secret, value)
		if err != nil {
			t.Fatalf("expect error be nil, got %v", err)
		}

		// a second call of 'Authorize' must use the cached secret value
		err, remoteCalled := auth.Authorize(ctx, secret, value)
		if err != nil {
			t.Fatalf("expect error be nil, got %v", err)
		}
		if spyCalls != 1 {
			t.Fatal("expect 'GetSecretValue' is called once", spyCalls)
		}
		if remoteCalled {
			t.Fatal("expect 'remoteCalled' be false, got true")
		}

		// wait until the cache is expired
		time.Sleep(ttl + 100*time.Millisecond)

		// a second secret API call has to be made
		err, remoteCalled = auth.Authorize(ctx, secret, value)
		if err != nil {
			t.Fatalf("expect error be nil, got %v", err)
		}
		if spyCalls != 2 {
			t.Fatal("expect 'GetSecretValue' is called twice")
		}
		if !remoteCalled {
			t.Fatal("expect 'remoteCalled' be true, got false")
		}
	})
}
