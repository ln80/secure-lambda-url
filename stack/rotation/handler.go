package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ln80/secure-lambda-url/cloudfront"
	"github.com/ln80/secure-lambda-url/secretsmanager"
)

type SecretsManagerRotationRequest struct {
	SecretID           string `json:"SecretId"`
	ClientRequestToken string `json:"ClientRequestToken"`
	Step               string `json:"Step"`
}

type handler func(context.Context, SecretsManagerRotationRequest) error

func makeHandler(distID, customHeader string, rotator secretsmanager.Rotator, updater cloudfront.Updater) handler {

	setSecret := func(ctx context.Context, current, pending string) error {
		if distID == "" || customHeader == "" {
			log.Println("WARNING: update dist origin ignored: missed dist ID or Header Name")
			return nil
		}

		return updater.Update(ctx, distID, cloudfront.UpdateCustomHeaderFn(customHeader, pending))
	}

	testSecret := func(ctx context.Context, pending string) error {
		// at the moment, we directly replace the CURRENT value by the PENDING.
		// TODO: figure out a simple way to test cloudfront dist custom header change before
		// finishing the rotation.
		return nil
	}

	return func(ctx context.Context, event SecretsManagerRotationRequest) (err error) {
		defer func() {
			if err != nil {
				log.Println("ERROR: rotation error occurred: ", err)
			}
		}()

		secret, token, step := event.SecretID, event.ClientRequestToken, event.Step

		if err = rotator.RotationEnabled(ctx, secret); err != nil {
			return err
		}

		switch step {
		case secretsmanager.StepCreate:
			err = rotator.Create(ctx, secret, token)
		case secretsmanager.StepSet:
			err = rotator.Set(ctx, secret, token, setSecret)
		case secretsmanager.StepTest:
			err = rotator.Test(ctx, secret, token, testSecret)
		case secretsmanager.StepFinish:
			err = rotator.Finish(ctx, secret, token)
		default:
			err = fmt.Errorf("%w: %s", secretsmanager.ErrRotationInvalidStep, step)
		}

		return
	}
}
