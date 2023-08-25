package secretsmanager

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

var (
	ErrInvalidSecretValue  = errors.New("invalid secret value")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrAuthorizationFailed = errors.New("authorization failed")
)

type Authorizer interface {
	Authorize(ctx context.Context, secretID, value string) error
}

type AuthorizerConfig struct {
	// gracePreriod is used to tolerate accepting "Previous" and "Pending" secret version
	// as valid values for a short period of time.
	GracePeriod time.Duration

	// coolDownPeriod is period during which we assume the secret can't be rotated.
	// It's used to rate limit the API calls
	CoolDownPeriod time.Duration
}

type DefaultAuthorizer struct {
	client ClientAPI

	janitor *Janitor
	cfg     *AuthorizerConfig
}

func NewAuthorizer(cli ClientAPI, j *Janitor, opts ...func(*AuthorizerConfig)) *DefaultAuthorizer {
	cfg := &AuthorizerConfig{
		GracePeriod:    15 * time.Second,
		CoolDownPeriod: 15 * time.Second,
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(cfg)
	}

	return &DefaultAuthorizer{
		client:  cli,
		cfg:     cfg,
		janitor: j,
	}
}

func (a *DefaultAuthorizer) Authorize(ctx context.Context, secretID, value string) error {
	if value == "" {
		return ErrInvalidSecretValue
	}
	if a.janitor.isBlackListed(value) {
		return ErrUnauthorized
	}

	cur, prev, pen, _ := a.janitor.getCache()
	defer func() {
		// refresh cache values
		a.janitor.setCache(cur, prev, pen)
	}()

	getSecret := func(stage string) (secret, error) {
		out, err := a.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
			SecretId:     aws.String(secretID),
			VersionStage: aws.String(stage),
		})

		s := secret{}
		if err != nil {
			var te *types.ResourceNotFoundException
			if errors.As(err, &te) {
				return s, nil
			}
			return s, fmt.Errorf("%w: %v", ErrAuthorizationFailed, err)
		}
		s.value = aws.ToString(out.SecretString)
		s.createdAt = aws.ToTime(out.CreatedDate)

		return s, nil
	}

	if cur.value == value {
		return nil
	}
	// only refresh secret cache value if cool down period is exceeded
	if time.Since(cur.createdAt) > a.cfg.CoolDownPeriod {
		var err error
		cur, err = getSecret(VersionCurrent)
		if err != nil {
			return err
		}
		if cur.value == value {
			return nil
		}
	}

	// Grace Period is a short and transitional period
	// during which checking auth against PREVIOUS and PENDING values is tolerated
	if time.Since(cur.createdAt) < a.cfg.GracePeriod {
		if time.Since(prev.createdAt) > a.cfg.CoolDownPeriod {
			var err error
			prev, err = getSecret(VersionPrevious)
			if err != nil {
				return err
			}
		}
		if prev.value == value {
			return nil
		}

		if time.Since(pen.createdAt) > a.cfg.CoolDownPeriod {
			var err error
			pen, err = getSecret(VersionPending)
			if err != nil {
				return err
			}
		}
		if pen.value == value {
			return nil
		}
	}

	a.janitor.blackList(value)

	return ErrUnauthorized
}
