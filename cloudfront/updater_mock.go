package cloudfront

import (
	"context"
)

// MockUpdater is a mock implementation of the Updater interface.
type MockUpdater struct {
	UpdateFn func(ctx context.Context, distID string, fns ...func(*DistributionConfig)) error
}

// Update mocks the Update method.
func (m *MockUpdater) Update(ctx context.Context, distID string, fns ...func(*DistributionConfig)) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, distID, fns...)
	}
	return nil
}
