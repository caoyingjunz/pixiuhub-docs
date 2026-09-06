package passwordutil

import "testing"

func TestEncryptAndValidatePassword(t *testing.T) {
	password := "S3cure-Passw0rd!"

	hash, err := EncryptPassword(password)
	if err != nil {
		t.Fatalf("EncryptPassword failed: %v", err)
	}
	if hash == password {
		t.Fatal("password stored as plaintext")
	}

	if err := ValidatePassword(hash, password); err != nil {
		t.Fatalf("ValidatePassword failed for correct password: %v", err)
	}

	if err := ValidatePassword(hash, "wrong-password"); err == nil {
		t.Fatal("expected error for wrong password")
	}
}

func TestValidatePasswordInvalidHash(t *testing.T) {
	if err := ValidatePassword("not-a-bcrypt-hash", "any"); err == nil {
		t.Fatal("expected error for invalid hash")
	}
}
