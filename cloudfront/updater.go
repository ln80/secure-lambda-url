package cloudfront

import (
	"context"
	"errors"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
)

var (
	ErrUpdateFunctionIsmissing = errors.New("update function is missing")
)

// DistributionConfig is an alias for "github.com/aws/aws-sdk-go-v2/service/cloudfront/types.DistributionConfig"
type DistributionConfig = types.DistributionConfig

// UpdateCustomHeaderFn returns a function that iterates over the distribution origins
// and updates the given custom header value if the header is found at the origin config level.
func UpdateCustomHeaderFn(headerName, headerValue string) func(*DistributionConfig) {
	headerName = http.CanonicalHeaderKey(headerName)

	return func(dc *DistributionConfig) {
		if dc.Origins == nil || len(dc.Origins.Items) == 0 {
			return
		}
		for i, origin := range dc.Origins.Items {
			if origin.CustomHeaders == nil || len(origin.CustomHeaders.Items) == 0 {
				continue
			}
			for j, h := range origin.CustomHeaders.Items {
				hname := http.CanonicalHeaderKey(aws.ToString(h.HeaderName))
				if hname == headerName {
					dc.Origins.Items[i].CustomHeaders.Items[j].HeaderValue = aws.String(headerValue)
					break
				}
			}
		}
	}
}

// Updater interface presents a service that updates a cloudfront distribution config.
type Updater interface {
	// Update fetches the distribution config and updates it using a set of functions.
	// An empty set of functions behavior is implementation-specific.
	Update(ctx context.Context, distID string, fns ...func(*DistributionConfig)) error
}

type DefaultUpdater struct {
	client ClientAPI
}

var _ Updater = &DefaultUpdater{}

func NewDefaultUpdater(cli ClientAPI) *DefaultUpdater {
	return &DefaultUpdater{client: cli}
}

// Update implements the Updater interface
func (u *DefaultUpdater) Update(ctx context.Context, distID string, fns ...func(*DistributionConfig)) error {
	// TODO: use the new generic slice function to filter nil values
	if len(fns) == 0 {
		return ErrUpdateFunctionIsmissing
	}

	out, err := u.client.GetDistributionConfig(ctx, &cloudfront.GetDistributionConfigInput{
		Id: aws.String(distID),
	})
	if err != nil {
		return err
	}
	distConfig := out.DistributionConfig

	for _, fn := range fns {
		if fn == nil {
			continue
		}
		fn(distConfig)
	}

	if _, err := u.client.UpdateDistribution(ctx, &cloudfront.UpdateDistributionInput{
		DistributionConfig: distConfig,
		Id:                 aws.String(distID),
		IfMatch:            out.ETag,
	}); err != nil {
		return err
	}

	return nil
}
