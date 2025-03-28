package backend

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"

	"github.com/charmbracelet/log/v2"
	"golang.org/x/crypto/bcrypt"
)

const saltySalt = "salty-soft-serve"

// HashPassword hashes the password using bcrypt.
func HashPassword(password string) (string, error) {
	crypt, err := bcrypt.GenerateFromPassword([]byte(password+saltySalt), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(crypt), nil
}

// VerifyPassword verifies the password against the hash.
func VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password+saltySalt))
	return err == nil
}

// GenerateToken returns a random unique token.
func GenerateToken() string {
	buf := make([]byte, 20)
	if _, err := rand.Read(buf); err != nil {
		log.Error("unable to generate access token")
		return ""
	}

	return "ss_" + hex.EncodeToString(buf)
}

// HashToken hashes the token using sha256.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token + saltySalt))
	return hex.EncodeToString(sum[:])
}
