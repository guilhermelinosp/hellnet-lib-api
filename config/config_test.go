package config

import (
	"strings"
	"testing"
	"time"
)

func TestWithValuesDefaults(t *testing.T) {
	cfg, err := WithValues(Values{Name: "svc"})
	if err != nil {
		t.Fatalf("WithValues errored: %v", err)
	}
	if cfg.Port != "8080" {
		t.Fatalf("port = %q, want 8080", cfg.Port)
	}
	if cfg.Env != "Development" {
		t.Fatalf("env = %q, want Development", cfg.Env)
	}
	if cfg.ReadTimeout != 15*time.Second {
		t.Fatalf("read timeout = %v", cfg.ReadTimeout)
	}
	if cfg.BodyLimit != 1<<20 {
		t.Fatalf("body limit = %d", cfg.BodyLimit)
	}
	if cfg.ReleaseMode {
		t.Fatal("Development should not be release mode")
	}
	if cfg.LogLevel.String() != "DEBUG" {
		t.Fatalf("log level = %s, want DEBUG in Development", cfg.LogLevel)
	}
}

func TestWithValuesExplicit(t *testing.T) {
	cfg, err := WithValues(Values{
		Name:           "api",
		Env:            "Production",
		Port:           "9090",
		BodyLimit:      4 << 20,
		TrustedProxies: []string{"10.0.0.0/8"},
	})
	if err != nil {
		t.Fatalf("WithValues errored: %v", err)
	}
	if cfg.Port != "9090" {
		t.Fatalf("port = %q", cfg.Port)
	}
	if !cfg.ReleaseMode {
		t.Fatal("Production should be release mode")
	}
	if len(cfg.TrustedProxies) != 1 || cfg.TrustedProxies[0] != "10.0.0.0/8" {
		t.Fatalf("trusted proxies = %v", cfg.TrustedProxies)
	}
}

func TestWithValuesValidation(t *testing.T) {
	_, err := WithValues(Values{Name: ""})
	if err == nil || !strings.Contains(err.Error(), "SERVICE") {
		t.Fatalf("expected SERVICE error, got %v", err)
	}
	_, err = WithValues(Values{Name: "x", Port: "notaport"})
	if err == nil || !strings.Contains(err.Error(), "PORT") {
		t.Fatalf("expected PORT error, got %v", err)
	}
	_, err = WithValues(Values{Name: "x", Port: "70000"})
	if err == nil {
		t.Fatal("expected out-of-range port error")
	}
}

func TestIsProduction(t *testing.T) {
	prod, _ := WithValues(Values{Name: "x", Env: "production"})
	if !prod.IsProduction() {
		t.Fatal("case-insensitive production should match")
	}
	dev, _ := WithValues(Values{Name: "x", Env: "development"})
	if dev.IsProduction() {
		t.Fatal("development should not be production")
	}
}
