package jwk

import (
	"crypto"
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"

	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/go-jose/go-jose/v3"
	"github.com/golang-jwt/jwt/v5"
)

// SigningMethod is a JSON Web Token signing method. It uses Ed25519 keys to
// sign and verify tokens.
var SigningMethod = &jwt.SigningMethodEd25519{}

// Pair is a JSON Web Key pair.
type Pair struct {
	privateKey crypto.PrivateKey
	jwk        jose.JSONWebKey
}

// PrivateKey returns the private key.
func (p Pair) PrivateKey() crypto.PrivateKey {
	return p.privateKey
}

// JWK returns the JSON Web Key.
func (p Pair) JWK() jose.JSONWebKey {
	return p.jwk
}

// NewPair creates a new JSON Web Key pair.
func NewPair(cfg *config.Config) (Pair, error) {
	kp, err := config.KeyPair(cfg)
	if err != nil {
		return Pair{}, err
	}

	// Derive kid from the public key bytes (Ed25519 public key is a []byte).
	var kidBytes []byte
	if pub, ok := kp.CryptoPublicKey().(ed25519.PublicKey); ok {
		kidBytes = pub
	} else {
		kidBytes = kp.RawPrivateKey()
	}
	kid := fmt.Sprintf("%x", sha256.Sum256(kidBytes))
	jwk := jose.JSONWebKey{
		Key:       kp.CryptoPublicKey(),
		KeyID:     kid,
		Algorithm: SigningMethod.Alg(),
	}

	return Pair{privateKey: kp.PrivateKey(), jwk: jwk}, nil
}
