package adapter

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/guilhermelinosp/hellnet-lib-api/api"
	"github.com/guilhermelinosp/hellnet-lib-api/config"
)

func TestTranslatePath(t *testing.T) {
	cases := map[string]struct {
		in   string
		want string
	}{
		"no placeholder": {"/health", "/health"},
		"single param":   {"/users/{id}", "/users/:id"},
		"two params":     {"/orgs/{oid}/repos/{rid}", "/orgs/:oid/repos/:rid"},
		"prefix group":   {"/api/v1", "/api/v1"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := TranslatePath(tc.in); got != tc.want {
				t.Fatalf("TranslatePath(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func newTestRouter(t *testing.T) *Router {
	t.Helper()
	cfg, err := config.WithValues(config.Values{Name: "api-test"})
	if err != nil {
		t.Fatalf("config.WithValues errored: %v", err)
	}
	return New(cfg, slog.Default())
}

func TestNewCORSDisabled(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected CORS header = %q", got)
	}
}

func TestNewCORSEnabled(t *testing.T) {
	cfg, err := config.WithValues(config.Values{Name: "api-test"})
	if err != nil {
		t.Fatalf("config.WithValues errored: %v", err)
	}
	cfg.CORSAllowedOrigins = []string{"https://app.example.com"}
	r := New(cfg, slog.Default())
	r.Handle(http.MethodGet, "/hello", api.HandlerFunc(func(_ context.Context, _ api.Request) (api.Response, error) {
		return api.JSON(http.StatusOK, map[string]string{"msg": "ok"}), nil
	}))
	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("allow-origin = %q", got)
	}
}

func TestNewRequestIDHeader(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if got := rec.Header().Get(RequestIDHeader); got == "" {
		t.Fatal("expected a request id header on the response")
	}
}

func TestJSONHandlerEmptyBody(t *testing.T) {
	r := newTestRouter(t)
	type echoIn struct {
		Name string `json:"name"`
	}
	r.Handle(http.MethodPost, "/echo", JSONHandler[echoIn, echoIn]{
		HandleFunc: func(_ context.Context, in *echoIn) (echoIn, error) {
			return echoIn{Name: in.Name}, nil
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/echo", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (empty body should be zero TReq)", rec.Code)
	}
}
