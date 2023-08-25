package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ln80/secure-lambda-url/secretsmanager"
)

// MakeHandler returns the http.Handler used by the sidecar process.
// Lambda handler will issue HTTP Get requests to this server for API key validation.
func MakeHandler(secretID, token string, auth secretsmanager.Authorizer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if t := r.Header.Get("X-Aws-Token"); t != token {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		k := strings.TrimSpace(r.URL.Query().Get("key"))

		if err := auth.Authorize(r.Context(), secretID, k); err != nil {
			if errors.Is(err, secretsmanager.ErrUnauthorized) {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}

// server is a simple wrapper on top of http.Server.
// It simplifies the start and the graceful shutdown of the http.server
type server struct {
	serv *http.Server
}

func NewServer(port string, h http.Handler) *server {
	mux := http.NewServeMux()
	mux.Handle("/", h)

	return &server{
		serv: &http.Server{
			Addr:    "127.0.0.1:" + port,
			Handler: mux,
		},
	}
}

// Start changes the default behavior of 'serve' method.
// It accepts a context, and allow to gracefully shutdown the server in context cancellation.
// A gracefully cancelled server does not return error as opposed to the default behavior.
func (s *server) Start(ctx context.Context) error {
	// Offload as many responsibilities as possible from the 'serve' method,
	// this make it simple to fail and return error
	l, err := net.Listen("tcp", s.serv.Addr)
	if err != nil {
		return err
	}

	var (
		closed bool
		mu     sync.Mutex
	)

	go func() {
		<-ctx.Done()
		mu.Lock()
		defer mu.Unlock()

		e := s.shutdown()
		if e == nil || errors.Is(e, http.ErrServerClosed) {
			closed = true
		}
	}()

	if err := s.serv.Serve(l); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			mu.Lock()
			defer mu.Unlock()

			if closed {
				return nil
			}
		}

		return err
	}

	return nil
}

func (s *server) shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	return s.serv.Shutdown(ctx)
}
