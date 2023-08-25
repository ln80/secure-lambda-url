package secretsmanager

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

func TestAPIClient(t *testing.T) {
	cfg, _ := config.LoadDefaultConfig(
		context.Background(),
	)
	cli := NewClient(cfg, "")

	_, ok := cli.(*secretsmanager.Client)
	if !ok {
		t.Fatalf("expect cli ins instance of %T", &secretsmanager.Client{})
	}
}
