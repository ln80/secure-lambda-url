package secretsmanager

import (
	"context"
)

// MockRotator is a mock implementation of the Rotator interface.
type MockRotator struct {
	RotationEnabledFn func(ctx context.Context, secretARN string) error
	CreateFn          func(ctx context.Context, secretARN, token string) error
	SetFn             func(ctx context.Context, secretARN, token string, fn func(ctx context.Context, current, pending string) error) error
	TestFn            func(ctx context.Context, secretARN, token string, fn func(ctx context.Context, pending string) error) error
	FinishFn          func(ctx context.Context, secretARN, token string) error
}

// RotationEnabled mocks the RotationEnabled method.
func (m *MockRotator) RotationEnabled(ctx context.Context, secretARN string) error {
	if m.RotationEnabledFn != nil {
		return m.RotationEnabledFn(ctx, secretARN)
	}
	return nil
}

// Create mocks the Create method.
func (m *MockRotator) Create(ctx context.Context, secretARN, token string) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, secretARN, token)
	}
	return nil
}

// Set mocks the Set method.
func (m *MockRotator) Set(ctx context.Context, secretARN, token string, fn func(ctx context.Context, current, pending string) error) error {
	if m.SetFn != nil {
		return m.SetFn(ctx, secretARN, token, fn)
	}
	return nil
}

// Test mocks the Test method.
func (m *MockRotator) Test(ctx context.Context, secretARN, token string, fn func(ctx context.Context, pending string) error) error {
	if m.TestFn != nil {
		return m.TestFn(ctx, secretARN, token, fn)
	}
	return nil
}

// Finish mocks the Finish method.
func (m *MockRotator) Finish(ctx context.Context, secretARN, token string) error {
	if m.FinishFn != nil {
		return m.FinishFn(ctx, secretARN, token)
	}
	return nil
}
