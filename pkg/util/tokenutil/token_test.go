package tokenutil

import "testing"

func TestGenerateAndParseLoginToken(t *testing.T) {
	key := []byte("test-jwt-key")
	token, err := GenerateLoginToken("user-001", "admin", 1, key)
	if err != nil {
		t.Fatalf("GenerateLoginToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("generated empty token")
	}

	claims, err := ParseLoginToken(token, key)
	if err != nil {
		t.Fatalf("ParseLoginToken failed: %v", err)
	}
	if claims.UserId != "user-001" {
		t.Fatalf("UserId mismatch: got %q want %q", claims.UserId, "user-001")
	}
	if claims.Name != "admin" {
		t.Fatalf("Name mismatch: got %q want %q", claims.Name, "admin")
	}
	if claims.Role != 1 {
		t.Fatalf("Role mismatch: got %d want %d", claims.Role, 1)
	}
}

func TestParseLoginTokenWrongKey(t *testing.T) {
	token, err := GenerateLoginToken("user-001", "admin", 1, []byte("correct-key"))
	if err != nil {
		t.Fatalf("GenerateLoginToken failed: %v", err)
	}
	if _, err := ParseLoginToken(token, []byte("wrong-key")); err == nil {
		t.Fatal("expected error when parsing with wrong key")
	}
}

func TestParseLoginTokenInvalid(t *testing.T) {
	if _, err := ParseLoginToken("invalid-token", []byte("key")); err == nil {
		t.Fatal("expected error for invalid token")
	}
}
