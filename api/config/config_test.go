package config

import (
	"strings"
	"testing"
)

func TestValidateRequiresTokenEncryptionKeyForTursoOrganization(t *testing.T) {
	err := Validate(Config{
		TursoOrganization: "acme",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "TOKEN_ENCRYPTION_KEY") {
		t.Fatalf("expected TOKEN_ENCRYPTION_KEY error, got %v", err)
	}
}

func TestValidateAllowsLocalConfigurationWithoutEncryptionKey(t *testing.T) {
	if err := Validate(Config{}); err != nil {
		t.Fatalf("expected local config to validate: %v", err)
	}
}

func TestValidateAllowsTursoOrganizationWithEncryptionKey(t *testing.T) {
	if err := Validate(Config{
		TursoOrganization:  "acme",
		TokenEncryptionKey: "0123456789abcdef0123456789abcdef",
	}); err != nil {
		t.Fatalf("expected Turso config with encryption key to validate: %v", err)
	}
}
