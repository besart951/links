package security

import "testing"

func TestPasswordHashAndVerify(t *testing.T) {
	hasher := NewArgon2idPasswordHasher()
	hash, err := hasher.Hash("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	ok, err := hasher.Verify("correct-horse-battery-staple", hash)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !ok {
		t.Fatal("expected password to verify")
	}

	ok, err = hasher.Verify("wrong-password", hash)
	if err != nil {
		t.Fatalf("Verify(wrong) error = %v", err)
	}
	if ok {
		t.Fatal("expected wrong password to fail")
	}
}

func TestHashPasswordRejectsShortPasswords(t *testing.T) {
	hasher := NewArgon2idPasswordHasher()
	if _, err := hasher.Hash("short"); err == nil {
		t.Fatal("expected short password to be rejected")
	}
}
