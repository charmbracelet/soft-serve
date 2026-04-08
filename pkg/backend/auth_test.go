package backend

import "testing"

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("password")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" {
		t.Fatal("hash is empty")
	}
}

func TestVerifyPassword(t *testing.T) {
	hash, err := HashPassword("password")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword("password", hash) {
		t.Fatal("password did not verify")
	}
}

func TestGenerateToken(t *testing.T) {
	token, err := GenerateToken()
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("token is empty")
	}
}

func TestHashToken(t *testing.T) {
	token, err := GenerateToken()
	if err != nil {
		t.Fatal(err)
	}
	hash := HashToken(token)
	if hash == "" {
		t.Fatal("hash is empty")
	}
}
