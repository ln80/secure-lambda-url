package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"golang.org/x/sync/errgroup"

	"github.com/ln80/secure-lambda-url/secretsmanager"
)

var (
	_unitTesting bool
)

var (
	extensionClient *client
	ipc             *server
	cache           *secretsmanager.Janitor
)

const (
	defaultPort = "3579"
)

func init() {
	if _unitTesting {
		return
	}
	secretEndpoint := os.Getenv("SECURE_LAMBDA_URL_SECRET_ENDPOINT")
	if secretEndpoint == "" {
		println("Init failed", fmt.Errorf(`
			missed env params:
			SECURE_LAMBDA_URL_SECRET_ENDPOINT: %s,
			`, secretEndpoint))
		os.Exit(1)
	}
	secretID := os.Getenv("SECURE_LAMBDA_URL_SECRET_ARN")
	if secretID == "" {
		println("Init failed", fmt.Errorf(`
			missed env params:
			SECURE_LAMBDA_URL_SECRET_ARN: %s,
			`, secretID))
		os.Exit(1)
	}

	cfg, err := config.LoadDefaultConfig(
		context.Background(),
	)
	if err != nil {
		println("Init failed", err)
		os.Exit(1)
	}

	port := os.Getenv("SECURE_LAMBDA_URL_HTTP_PORT")
	if port == "" {
		port = defaultPort
	}

	cache = secretsmanager.NewJanitor(20 * time.Minute)

	ipc = NewServer(
		port,
		MakeHandler(secretID, os.Getenv("AWS_SESSION_TOKEN"),
			secretsmanager.NewAuthorizer(secretsmanager.NewClient(cfg, secretEndpoint), cache),
		),
	)

	extensionClient = NewClient(os.Getenv("AWS_LAMBDA_RUNTIME_API"))
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)

	cache.Run(ctx, func() {
		println("Secret cache cleared")
	})

	// Extension has to terminate if either client or IPC server has terminated

	// Extension has to exit with error if:
	// - IPC server stopped for a reason other than context cancellation
	// - Extension failed to register
	// - Extension failed to receive next event

	// TDB: use errgroup with context instead
	g := &errgroup.Group{}

	g.Go(func() error {
		defer cancel()
		return ipc.Start(ctx)
	})

	// A Shameless hack to give IPC server a chance to start before registering the extension.
	// Otherwise, lambda runtime might prematurely receive events and call the IPC server.
	// TBD: sleep duration
	time.Sleep(10 * time.Millisecond)
	if err := registerExtension(ctx); err != nil {
		println("Register failed", err)
		os.Exit(1)
	}

	g.Go(func() error {
		defer cancel()
		return processEvents(ctx)
	})

	if err := g.Wait(); err != nil {
		println("Exiting with failure...", err)
		os.Exit(1)
	}
}

func registerExtension(ctx context.Context) error {
	res, err := extensionClient.Register(ctx, extensionName)
	if err != nil {
		return err
	}
	println("Register response:", prettyPrint(res))

	return nil
}

func processEvents(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			println("Waiting for event...")
			res, err := extensionClient.NextEvent(ctx)
			if err != nil {
				println("Error:", err)
				return err
			}
			println("Received event:", prettyPrint(res))
			// Exit if we receive a SHUTDOWN event
			if res.EventType == Shutdown {
				println("Received SHUTDOWN event")
				return nil
			}
		}
	}
}
