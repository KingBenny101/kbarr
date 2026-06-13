package auth

import "testing"

func TestEnvAuth(t *testing.T) {
	t.Run("inactive when password unset", func(t *testing.T) {
		t.Setenv("KBARR_AUTH_USERNAME", "someone")
		t.Setenv("KBARR_AUTH_PASSWORD", "")
		if _, _, active := EnvAuth(); active {
			t.Fatal("expected override inactive when password is empty")
		}
	})

	t.Run("username defaults to admin", func(t *testing.T) {
		t.Setenv("KBARR_AUTH_USERNAME", "")
		t.Setenv("KBARR_AUTH_PASSWORD", "secret")
		user, pass, active := EnvAuth()
		if !active || user != "admin" || pass != "secret" {
			t.Fatalf("got (%q, %q, %v), want (admin, secret, true)", user, pass, active)
		}
	})
}

func TestValidateCredentialsEnvOverride(t *testing.T) {
	t.Setenv("KBARR_AUTH_USERNAME", "operator")
	t.Setenv("KBARR_AUTH_PASSWORD", "hunter2")

	// A nil DB proves the stored hash is never consulted while the override is on.
	if !ValidateCredentials(nil, "operator", "hunter2") {
		t.Error("correct env credentials should validate")
	}
	if !ValidateCredentials(nil, "OPERATOR", "hunter2") {
		t.Error("username comparison should be case-insensitive")
	}
	if ValidateCredentials(nil, "operator", "wrong") {
		t.Error("wrong password must fail")
	}
	if ValidateCredentials(nil, "intruder", "hunter2") {
		t.Error("wrong username must fail")
	}
}

func TestUpdateCredentialsRejectedWhenEnvManaged(t *testing.T) {
	t.Setenv("KBARR_AUTH_PASSWORD", "hunter2")
	err := UpdateCredentials(nil, "hunter2", "new", "newpass")
	if _, ok := err.(*ErrEnvManaged); !ok {
		t.Fatalf("expected ErrEnvManaged, got %v", err)
	}
}
