package auth

import (
	"errors"
	"testing"
)

func TestMemorySecretStore(t *testing.T) {
	store := NewMemorySecretStore()

	if _, err := store.Get(KeyringService, InvestCredentials); !errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("Get missing err = %v, want ErrSecretNotFound", err)
	}

	if err := store.Set(KeyringService, InvestCredentials, "secret"); err != nil {
		t.Fatalf("Set err = %v", err)
	}

	got, err := store.Get(KeyringService, InvestCredentials)
	if err != nil {
		t.Fatalf("Get err = %v", err)
	}
	if got != "secret" {
		t.Fatalf("Get = %q, want %q", got, "secret")
	}

	if err := store.Delete(KeyringService, InvestCredentials); err != nil {
		t.Fatalf("Delete err = %v", err)
	}

	if _, err := store.Get(KeyringService, InvestCredentials); !errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("Get after delete err = %v, want ErrSecretNotFound", err)
	}
}
