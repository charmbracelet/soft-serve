package web

import (
	"context"
	"crypto"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"charm.land/log/v2"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/golang-jwt/jwt/v5"
)

// jwtPublicKeyCache caches the parsed Ed25519 public key per SSH key path.
// Avoids reading and parsing the key file on every JWT Bearer request.
var jwtPublicKeyCache sync.Map // map[string]crypto.PublicKey

func cachedJWTPublicKey(cfg *config.Config) (crypto.PublicKey, error) {
	if v, ok := jwtPublicKeyCache.Load(cfg.SSH.KeyPath); ok {
		return v.(crypto.PublicKey), nil
	}
	kp, err := config.KeyPair(cfg)
	if err != nil {
		return nil, err
	}
	pub := kp.CryptoPublicKey()
	jwtPublicKeyCache.Store(cfg.SSH.KeyPath, pub)
	return pub, nil
}

// authenticate authenticates the user from the request.
func authenticate(r *http.Request) (proto.User, error) {
	// Prefer the Authorization header
	user, err := parseAuthHdr(r)
	if err != nil || user == nil {
		if errors.Is(err, ErrInvalidToken) || errors.Is(err, ErrInvalidPassword) {
			return nil, err
		}
		return nil, proto.ErrUserNotFound
	}

	return user, nil
}

// ErrInvalidPassword is returned when the password is invalid.
var ErrInvalidPassword = errors.New("invalid password")

func parseUsernamePassword(ctx context.Context, username, password string) (proto.User, error) {
	logger := log.FromContext(ctx)
	be := backend.FromContext(ctx)

	if username != "" && password != "" {
		user, err := be.User(ctx, username)
		if err == nil && user != nil && backend.VerifyPassword(password, user.Password()) {
			return user, nil
		}

		// Try to authenticate using access token as the password
		user, err = be.UserByAccessToken(ctx, password)
		if err == nil {
			return user, nil
		}

		logger.Error("invalid password or token", "username", username, "err", err)
		return nil, ErrInvalidPassword
	} else if username != "" {
		// Try to authenticate using access token as the username
		logger.Debug("trying to authenticate using access token as username", "username", username)
		user, err := be.UserByAccessToken(ctx, username)
		if err == nil {
			return user, nil
		}

		logger.Error("failed to get user", "err", err)
		return nil, ErrInvalidToken
	}

	return nil, proto.ErrUserNotFound
}

// ErrInvalidHeader is returned when the authorization header is invalid.
var ErrInvalidHeader = errors.New("invalid authorization header")

func parseAuthHdr(r *http.Request) (proto.User, error) {
	// Check for auth header
	header := r.Header.Get("Authorization")
	if header == "" {
		return nil, ErrInvalidHeader
	}

	ctx := r.Context()
	logger := log.FromContext(ctx).WithPrefix("http.auth")
	be := backend.FromContext(ctx)

	logger.Debug("authorization auth header", "header", header)

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid authorization header")
	}

	switch strings.ToLower(parts[0]) {
	case "token":
		user, err := be.UserByAccessToken(ctx, parts[1])
		if err != nil {
			logger.Error("failed to get user", "err", err)
			return nil, err
		}

		return user, nil
	case "bearer":
		claims, err := parseJWT(ctx, parts[1])
		if err != nil {
			return nil, err
		}

		// Find the user
		parts := strings.SplitN(claims.Subject, "#", 2)
		if len(parts) != 2 {
			logger.Error("invalid jwt subject", "subject", claims.Subject)
			return nil, errors.New("invalid jwt subject")
		}

		user, err := be.User(ctx, parts[0])
		if err != nil {
			logger.Error("failed to get user", "err", err)
			return nil, err
		}

		expectedSubject := fmt.Sprintf("%s#%d", user.Username(), user.ID())
		if expectedSubject != claims.Subject {
			logger.Error("invalid jwt subject", "subject", claims.Subject, "expected", expectedSubject)
			return nil, errors.New("invalid jwt subject")
		}

		return user, nil
	default:
		username, password, ok := r.BasicAuth()
		if !ok {
			return nil, ErrInvalidHeader
		}

		return parseUsernamePassword(ctx, username, password)
	}
}

// ErrInvalidToken is returned when a token is invalid.
var ErrInvalidToken = errors.New("invalid token")

func parseJWT(ctx context.Context, bearer string) (*jwt.RegisteredClaims, error) {
	cfg := config.FromContext(ctx)
	logger := log.FromContext(ctx).WithPrefix("http.auth")
	pub, err := cachedJWTPublicKey(cfg)
	if err != nil {
		return nil, err
	}

	repo := proto.RepositoryFromContext(ctx)
	if repo == nil {
		return nil, errors.New("missing repository")
	}

	token, err := jwt.ParseWithClaims(bearer, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, errors.New("invalid signing method")
		}

		return pub, nil
	},
		jwt.WithIssuer(cfg.HTTP.PublicURL),
		jwt.WithIssuedAt(),
		jwt.WithAudience(repo.Name()),
	)
	if err != nil {
		logger.Error("failed to parse jwt", "err", err)
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !token.Valid || !ok {
		return nil, ErrInvalidToken
	}

	// Validate JWT claims before accepting.
	// Prevents not-before, issuer, and audience attacks.
	if err := validateJWTClaims(ctx, cfg, claims); err != nil {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// validateJWTClaims validates JWT claims for security.
// Prevents not-before, issuer, and audience attacks.
func validateJWTClaims(ctx context.Context, cfg *config.Config, claims *jwt.RegisteredClaims) error {
	// Validate expiration time if set
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return proto.ErrTokenExpired
	}

	// Validate not-before time if set
	if claims.NotBefore != nil && claims.NotBefore.Time.After(time.Now()) {
		return errors.New("token not yet valid")
	}

	// Validate issuer
	if claims.Issuer != cfg.HTTP.PublicURL {
		return errors.New("invalid token issuer")
	}

	// Audience is validated by jwt.WithAudience(repo.Name()) in ParseWithClaims.
	// No additional audience check is needed here.

	return nil
}
