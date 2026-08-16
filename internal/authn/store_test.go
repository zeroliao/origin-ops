package authn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStoreHashesPasswordsAndInvalidatesPriorRevision(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth", "users.json")
	store := NewStore(path)
	if err := store.Add("operator", "initial-password"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "initial-password") {
		t.Fatal("credential file contains the plaintext password")
	}
	principal, valid, err := store.Authenticate("operator", "initial-password")
	if err != nil || !valid {
		t.Fatalf("authenticate = %+v, %v, %v", principal, valid, err)
	}
	if err := store.SetPassword("operator", "replacement-password"); err != nil {
		t.Fatal(err)
	}
	validPrincipal, err := store.Validate(principal)
	if err != nil {
		t.Fatal(err)
	}
	if validPrincipal {
		t.Fatal("password change did not invalidate the previous revision")
	}
	if _, valid, err := store.Authenticate("operator", "initial-password"); err != nil || valid {
		t.Fatalf("old password remained valid: %v, %v", valid, err)
	}
	if _, valid, err := store.Authenticate("operator", "replacement-password"); err != nil || !valid {
		t.Fatalf("replacement password was rejected: %v, %v", valid, err)
	}
}

func TestManagerRejectsSessionAfterUserDisabled(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "users.json"))
	if err := store.Add("operator", "initial-password"); err != nil {
		t.Fatal(err)
	}
	manager := NewManager(store, time.Hour)
	token, _, _, valid, err := manager.Login("operator", "initial-password")
	if err != nil || !valid {
		t.Fatalf("login failed: %v, %v", valid, err)
	}
	if err := store.SetEnabled("operator", false); err != nil {
		t.Fatal(err)
	}
	if _, authenticated, err := manager.Current(token); err != nil || authenticated {
		t.Fatalf("disabled user retained session: %v, %v", authenticated, err)
	}
}
