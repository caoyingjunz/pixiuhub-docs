package encryptutil

import "testing"

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := DeriveKey("test-secret-key")
	plaintext := "my-sensitive-secret"

	enc, err := Encrypt([]byte(plaintext), key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if enc == plaintext {
		t.Fatal("Encrypt returned plaintext")
	}

	dec, err := Decrypt(enc, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if dec != plaintext {
		t.Fatalf("round trip mismatch: got %q want %q", dec, plaintext)
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key := DeriveKey("secret-a")
	wrongKey := DeriveKey("secret-b")

	enc, err := Encrypt([]byte("data"), key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if _, err := Decrypt(enc, wrongKey); err == nil {
		t.Fatal("expected error when decrypting with wrong key")
	}
}

func TestDecryptInvalidInput(t *testing.T) {
	key := DeriveKey("test")
	if _, err := Decrypt("not-base64-!!!", key); err == nil {
		t.Fatal("expected error for invalid base64")
	}
}
