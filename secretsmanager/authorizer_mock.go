package secretsmanager

import "context"

// MockAuthorizer is a mock implementation of the Updater interface.
type MockAuthorizer struct {
	AuthorizeFn func(ctx context.Context, secretID, value string) (error, bool)
}

var _ Authorizer = &MockAuthorizer{}

// Update mocks the Update method.
func (m *MockAuthorizer) Authorize(ctx context.Context, secretID, value string) (error, bool) {
	if m.AuthorizeFn != nil {
		return m.AuthorizeFn(ctx, secretID, value)
	}
	return nil, false
}
