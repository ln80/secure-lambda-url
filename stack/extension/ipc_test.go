package main

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ln80/secure-lambda-url/secretsmanager"
	"golang.org/x/sync/errgroup"
)

var _ = (func() interface{} {
	_unitTesting = true
	return nil
}())

func TestServer(t *testing.T) {
	randomPort := func() string {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			panic("unable to listen on a local address: " + err.Error())
		}
		addr := l.Addr().(*net.TCPAddr)
		l.Close()

		return strconv.Itoa(addr.Port)
	}

	ctx := context.Background()

	t.Run("is reachable", func(t *testing.T) {
		port := randomPort()
		spyCalls := int32(0)

		s := NewServer(port, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&spyCalls, 1)
		}))

		g := &errgroup.Group{}

		time.AfterFunc(2*time.Second, func() {
			t.Fatal("time out")
		})

		g.Go(func() error {
			_ = s.Start(ctx)
			return nil
		})

		g.Go(func() error {
			defer func() {
				_ = s.shutdown()
			}()

			_, err := http.Get("http://localhost:" + port)
			if err != nil {
				t.Fatal("expect err be nil, got", err)
			}

			if count := atomic.LoadInt32(&spyCalls); count != 1 {
				t.Fatalf("expect server to serve once, got: %d", count)
			}

			return nil
		})

		_ = g.Wait()
	})

	t.Run("is closable", func(t *testing.T) {
		port := randomPort()

		s := NewServer(port, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

		ctx, cancel := context.WithCancel(ctx)

		g := &errgroup.Group{}

		time.AfterFunc(2*time.Second, func() {
			t.Fatal("time out")
		})

		g.Go(func() error {
			time.Sleep(100 * time.Millisecond)
			cancel()
			return nil
		})

		g.Go(func() error {
			err := s.Start(ctx)
			if err != nil {
				t.Fatal("expect err be nil, got", err)
			}

			return nil
		})

		_ = g.Wait()
	})

	t.Run("with authorize handler", func(t *testing.T) {
		port := randomPort()
		token := "random"
		secret := "random"
		authMock := &secretsmanager.MockAuthorizer{}

		h := MakeHandler(secret, token, authMock)

		s := NewServer(port, h)

		g := &errgroup.Group{}

		time.AfterFunc(2*time.Second, func() {
			t.Fatal("time out")
		})

		g.Go(func() error {
			_ = s.Start(ctx)

			return nil
		})

		g.Go(func() error {
			defer func() {
				_ = s.shutdown()
			}()

			client := &http.Client{}

			// test invalid request
			req, _ := http.NewRequest("GET", "http://localhost:"+port, nil)
			r, _ := client.Do(req)
			r.Body.Close()
			if want, got := 400, r.StatusCode; want != got {
				t.Fatalf("expect %d, %d be equals", want, got)
			}

			// test unauthorized request
			authMock.AuthorizeFn = func(ctx context.Context, secretID, value string) (error, bool) {
				return secretsmanager.ErrUnauthorized, false
			}
			req, _ = http.NewRequest("GET", "http://localhost:"+port+"/?key=xyz", nil)
			req.Header.Add("X-Aws-Token", token)
			r, _ = client.Do(req)
			r.Body.Close()
			if want, got := 401, r.StatusCode; want != got {
				t.Fatalf("expect %d, %d be equals", want, got)
			}

			// test unexpected authorizer failed request
			authMock.AuthorizeFn = func(ctx context.Context, secretID, value string) (error, bool) {
				return secretsmanager.ErrAuthorizationFailed, false
			}
			req, _ = http.NewRequest("GET", "http://localhost:"+port+"/?key=xyz", nil)
			req.Header.Add("X-Aws-Token", token)
			r, _ = client.Do(req)
			r.Body.Close()
			if want, got := 500, r.StatusCode; want != got {
				t.Fatalf("expect %d, %d be equals", want, got)
			}

			return nil
		})

		_ = g.Wait()
	})
}
