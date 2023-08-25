package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/ln80/secure-lambda-url/cloudfront"
	"github.com/ln80/secure-lambda-url/secretsmanager"
)

var (
	_unitTesting bool
)

var (
	updater cloudfront.Updater
	rotator secretsmanager.Rotator
)

func init() {
	if _unitTesting {
		return
	}

	secretEndpoint := os.Getenv("SECRETS_MANAGER_ENDPOINT")
	if secretEndpoint == "" {
		log.Fatalf(`
			missed env params:
			SECRETS_MANAGER_ENDPOINT: %s,
			`, secretEndpoint)
	}

	cfg, err := config.LoadDefaultConfig(
		context.Background(),
	)
	if err != nil {
		log.Fatalln(err, "init dependencies failed")
	}

	rotator = secretsmanager.NewDefaultRotator(
		secretsmanager.NewClient(cfg, secretEndpoint))

	updater = cloudfront.NewDefaultUpdater(
		cloudfront.NewClient(cfg))
}

func main() {
	h := makeHandler(
		os.Getenv("DISTRIBUTION_ID"), os.Getenv("CUSTOM_HEADER_NAME"), rotator, updater)

	lambda.Start(h)
}
