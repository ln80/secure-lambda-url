package main

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/ln80/secure-lambda-url/cloudfront"
	"github.com/ln80/secure-lambda-url/secretsmanager"
)

var _ = (func() interface{} {
	_unitTesting = true
	return nil
}())

func TestHandler(t *testing.T) {

	ctx := context.Background()

	type tc struct {
		dist, customHeader string

		updater cloudfront.Updater
		rotator secretsmanager.Rotator

		evt SecretsManagerRotationRequest

		ok  bool
		err error
	}

	tcs := []tc{
		// invalid event step
		{
			rotator: &secretsmanager.MockRotator{
				RotationEnabledFn: func(ctx context.Context, secretARN string) error {
					return nil
				},
			},
			evt: SecretsManagerRotationRequest{
				SecretID:           "random",
				ClientRequestToken: "random",
				Step:               "invalid_step",
			},
			ok:  false,
			err: secretsmanager.ErrRotationInvalidStep,
		},
		// valid event but rotation is disabled
		{
			rotator: &secretsmanager.MockRotator{
				RotationEnabledFn: func(ctx context.Context, secretARN string) error {
					return secretsmanager.ErrRotationDisabled
				},
			},
			evt: SecretsManagerRotationRequest{
				SecretID:           "random",
				ClientRequestToken: "random",
				Step:               secretsmanager.StepCreate,
			},
			ok:  false,
			err: secretsmanager.ErrRotationDisabled,
		},
		// unexpected error during create step
		func() tc {
			infraErr := errors.New("infra error")
			return tc{
				rotator: &secretsmanager.MockRotator{
					RotationEnabledFn: func(ctx context.Context, secretARN string) error {
						return nil
					},
					CreateFn: func(ctx context.Context, secretARN, token string) error {
						return infraErr
					}},
				evt: SecretsManagerRotationRequest{
					SecretID:           "random",
					ClientRequestToken: "random",
					Step:               secretsmanager.StepCreate,
				},
				ok:  false,
				err: infraErr,
			}
		}(),
		// cloudfront update ignored if distID or CustomHeader is missing
		func() tc {
			return tc{
				updater: &cloudfront.MockUpdater{
					UpdateFn: func(ctx context.Context, distID string, fns ...func(*cloudfront.DistributionConfig)) error {
						return errors.New("unwanted error, should not be returned")
					},
				},
				rotator: &secretsmanager.MockRotator{
					RotationEnabledFn: func(ctx context.Context, secretARN string) error {
						return nil
					},
					SetFn: func(ctx context.Context, secretARN, token string, fn func(ctx context.Context, current, pending string) error) error {
						// a necessary shallow logic to wire updater into exec process
						return fn(ctx, "cur", "pen")
					},
				},
				evt: SecretsManagerRotationRequest{
					SecretID:           "random",
					ClientRequestToken: "random",
					Step:               secretsmanager.StepSet,
				},
				ok:  true,
				err: nil,
			}
		}(),
		// cloudfront update failed due to infra error
		func() tc {
			infraErr := errors.New("infra error")
			return tc{
				dist:         "random",
				customHeader: "X-Random",
				updater: &cloudfront.MockUpdater{
					UpdateFn: func(ctx context.Context, distID string, fns ...func(*cloudfront.DistributionConfig)) error {
						return infraErr
					},
				},
				rotator: &secretsmanager.MockRotator{
					RotationEnabledFn: func(ctx context.Context, secretARN string) error {
						return nil
					},
					SetFn: func(ctx context.Context, secretARN, token string, fn func(ctx context.Context, current, pending string) error) error {
						// a necessary shallow logic to wire updater into exec process
						return fn(ctx, "cur", "pen")
					},
				},
				evt: SecretsManagerRotationRequest{
					SecretID:           "random",
					ClientRequestToken: "random",
					Step:               secretsmanager.StepSet,
				},
				ok:  false,
				err: infraErr,
			}
		}(),
		// rotation set step succeed
		func() tc {
			return tc{
				dist:         "random",
				customHeader: "X-Random",
				updater: &cloudfront.MockUpdater{
					UpdateFn: func(ctx context.Context, distID string, fns ...func(*cloudfront.DistributionConfig)) error {
						return nil
					},
				},
				rotator: &secretsmanager.MockRotator{
					RotationEnabledFn: func(ctx context.Context, secretARN string) error {
						return nil
					},
					SetFn: func(ctx context.Context, secretARN, token string, fn func(ctx context.Context, current, pending string) error) error {
						// a necessary shallow logic to wire updater into exec process
						return fn(ctx, "cur", "pen")
					},
				},
				evt: SecretsManagerRotationRequest{
					SecretID:           "random",
					ClientRequestToken: "random",
					Step:               secretsmanager.StepSet,
				},
				ok:  true,
				err: nil,
			}
		}(),
		// rotation test step succeed
		func() tc {
			return tc{
				rotator: &secretsmanager.MockRotator{
					RotationEnabledFn: func(ctx context.Context, secretARN string) error {
						return nil
					},
					TestFn: func(ctx context.Context, secretARN, token string, fn func(ctx context.Context, pending string) error) error {
						// a necessary shallow logic to wire updater into exec process
						return fn(ctx, "pen")
					},
				},
				evt: SecretsManagerRotationRequest{
					SecretID:           "random",
					ClientRequestToken: "random",
					Step:               secretsmanager.StepTest,
				},
				ok:  true,
				err: nil,
			}
		}(),
		// rotation finish step failed due infra error
		func() tc {
			infraErr := errors.New("infra error")
			return tc{
				rotator: &secretsmanager.MockRotator{
					RotationEnabledFn: func(ctx context.Context, secretARN string) error {
						return nil
					},
					FinishFn: func(ctx context.Context, secretARN, token string) error {
						return infraErr
					},
				},
				evt: SecretsManagerRotationRequest{
					SecretID:           "random",
					ClientRequestToken: "random",
					Step:               secretsmanager.StepFinish,
				},
				ok:  false,
				err: infraErr,
			}
		}(),
		// rotation finish step succeed
		func() tc {
			return tc{
				rotator: &secretsmanager.MockRotator{
					RotationEnabledFn: func(ctx context.Context, secretARN string) error {
						return nil
					},
					FinishFn: func(ctx context.Context, secretARN, token string) error {
						return nil
					},
				},
				evt: SecretsManagerRotationRequest{
					SecretID:           "random",
					ClientRequestToken: "random",
					Step:               secretsmanager.StepFinish,
				},
				ok:  true,
				err: nil,
			}
		}(),
	}

	for i, tc := range tcs {
		t.Run("tc: "+strconv.Itoa(i+1), func(t *testing.T) {
			h := makeHandler(tc.dist, tc.customHeader, tc.rotator, tc.updater)
			err := h(ctx, tc.evt)
			if tc.ok {
				if err != nil {
					t.Fatal("expect err be nil, got", err)
				}
			} else {
				if !errors.Is(err, tc.err) {
					t.Fatalf("expect err be %v, got %v", tc.err, err)
				}
			}
		})
	}
}
