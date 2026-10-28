package config

import (
	"testing"
)

func TestLoadConfig_WithServiceRoleKeyAndCustomEnv(t *testing.T) {
	t.Setenv("NEXT_PUBLIC_SUPABASE_URL", "https://example.supabase.co")
	t.Setenv("NEXT_PUBLIC_SUPABASE_SERVICE_ROLE_KEY", "service-role-secret-key")
	t.Setenv("PORT", "9090")
	t.Setenv("CORS_ORIGIN", "http://localhost:3000")

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
	if len(cfg.JWTSecret) == 0 {
		t.Error("expected non-empty JWTSecret")
	}
}

func TestLoadConfig_WithPublishableKeyAndDefaults(t *testing.T) {
	t.Setenv("NEXT_PUBLIC_SUPABASE_URL", "https://example.supabase.co")
	t.Setenv("NEXT_PUBLIC_SUPABASE_SERVICE_ROLE_KEY", "")
	t.Setenv("NEXT_PUBLIC_SUPABASE_PUBLISHABLE_KEY", "anon-public-key")
	t.Setenv("PORT", "")
	t.Setenv("CORS_ORIGIN", "")

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
}
