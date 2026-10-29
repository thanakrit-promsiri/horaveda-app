package config

import (
	"testing"
)

func TestLoadConfig_WithServiceRoleKeyAndCustomEnv(t *testing.T) {
	t.Setenv("NEXT_PUBLIC_SUPABASE_URL", "https://example.supabase.co")
	t.Setenv("NEXT_PUBLIC_SUPABASE_SERVICE_ROLE_KEY", "service-role-secret-key")
	t.Setenv("PORT", "9090")
	t.Setenv("CORS_ORIGIN", "http://localhost:3000")
	t.Setenv("JWT_SECRET", "custom-jwt-secret")
	t.Setenv("COOKIE_SECURE", "true")

	cfg := LoadConfig()
	if cfg == nil {
		t.Fatal("expected non-nil Config")
	}
	if cfg.SupabaseClient == nil {
		t.Error("expected non-nil SupabaseClient")
	}
	if cfg.Logger == nil {
		t.Error("expected non-nil Logger")
	}
	if cfg.Port != "9090" {
		t.Errorf("expected Port 9090, got %q", cfg.Port)
	}
	if cfg.CORSOrigin != "http://localhost:3000" {
		t.Errorf("expected CORSOrigin http://localhost:3000, got %q", cfg.CORSOrigin)
	}
	if string(cfg.JWTSecret) != "custom-jwt-secret" {
		t.Errorf("expected JWTSecret custom-jwt-secret, got %q", string(cfg.JWTSecret))
	}
	if !cfg.CookieSecure {
		t.Error("expected CookieSecure=true")
	}
}

func TestLoadConfig_WithPublishableKeyAndDefaults(t *testing.T) {
	t.Setenv("NEXT_PUBLIC_SUPABASE_URL", "https://example.supabase.co")
	t.Setenv("NEXT_PUBLIC_SUPABASE_SERVICE_ROLE_KEY", "")
	t.Setenv("NEXT_PUBLIC_SUPABASE_PUBLISHABLE_KEY", "anon-public-key")
	t.Setenv("PORT", "")
	t.Setenv("CORS_ORIGIN", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("COOKIE_SECURE", "")
	t.Setenv("APP_ENV", "")

	cfg := LoadConfig()
	if cfg == nil {
		t.Fatal("expected non-nil Config")
	}
	if cfg.Port != "8080" {
		t.Errorf("expected default Port 8080, got %q", cfg.Port)
	}
	if cfg.CORSOrigin != "https://hora.garbagecollection.info" {
		t.Errorf("expected default CORSOrigin, got %q", cfg.CORSOrigin)
	}
	if cfg.CookieSecure {
		t.Error("expected default CookieSecure=false")
	}
}
