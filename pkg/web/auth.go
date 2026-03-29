package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"charm.land/log/v2"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// authenticate authenticates the user from the request.
func authenticate(r *http.Request) (proto.User, error) {
	// Prefer the Authorization header
	user, err := parseAuthHdr(r)
	if err != nil || user == nil {
		if errors.Is(err, errInvalidToken) || errors.Is(err, errInvalidPassword) {
			return nil, err
		}
		return nil, proto.ErrUserNotFound
	}

	return user, nil
}

// errInvalidPassword is returned when the password is invalid.
var errInvalidPassword = errors.New("invalid password")

// dummyHash is a bcrypt hash used to equalize timing when user doesn't exist.
// This prevents username enumeration via timing differences.
const dummyHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

func parseUsernamePassword(ctx context.Context, username, password string) (proto.User, error) {
	logger := log.FromContext(ctx)
	be := backend.FromContext(ctx)

	if username != "" && password != "" {
		user, err := be.User(ctx, username)
		if err != nil {
			// Run a dummy bcrypt comparison to prevent username enumeration via timing.
			_ = bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte(password))
			// Timing equalization is best-effort. The DB query has variable latency,
			// so this is not a constant-time guarantee but reduces the timing gap.
			// Also attempt token lookup so the not-found and wrong-password paths
			// do the same amount of work.
			_, _ = be.UserByAccessToken(ctx, password)
			return nil, errInvalidPassword
		}
		if user != nil && backend.VerifyPassword(password, user.Password()) {
			return user, nil
		}

		// Try to authenticate using access token as the password
		user, err = be.UserByAccessToken(ctx, password)
		if err == nil {
			return user, nil
		}

		logUsername := username
		if len(logUsername) > 20 {
			logUsername = logUsername[:20] + "…"
		}
		logger.Debug("invalid credentials", "username", logUsername, "err", err)
		return nil, errInvalidPassword
	} else if username != "" {
		// Try to authenticate using access token as the username
		logUser := username
		if len(logUser) > 8 {
			logUser = logUser[:8] + "…"
		}
		logger.Debug("trying to authenticate using access token as username", "username", logUser)
		user, err := be.UserByAccessToken(ctx, username)
		if err == nil {
			return user, nil
		}

		logger.Error("failed to get user", "err", err)
		return nil, errInvalidToken
	}

	return nil, proto.ErrUserNotFound
}

// errInvalidHeader is returned when the authorization header is invalid.
var errInvalidHeader = errors.New("invalid authorization header")

func parseAuthHdr(r *http.Request) (proto.User, error) {
	// Check for auth header
	header := r.Header.Get("Authorization")
	if header == "" {
		return nil, errInvalidHeader
	}

	ctx := r.Context()
	logger := log.FromContext(ctx).WithPrefix("http.auth")
	be := backend.FromContext(ctx)

	logger.Debug("authorization auth header", "scheme", strings.SplitN(header, " ", 2)[0])

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
			return nil, errInvalidHeader
		}

		return parseUsernamePassword(ctx, username, password)
	}
}

// errInvalidToken is returned when a token is invalid.
var errInvalidToken = errors.New("invalid token")

func parseJWT(ctx context.Context, bearer string) (*jwt.RegisteredClaims, error) {
	cfg := config.FromContext(ctx)
	logger := log.FromContext(ctx).WithPrefix("http.auth")
	kp, err := config.KeyPair(cfg)
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

		return kp.CryptoPublicKey(), nil
	},
		jwt.WithIssuer(cfg.HTTP.PublicURL),
		jwt.WithIssuedAt(),
		jwt.WithAudience(repo.Name()),
	)
	if err != nil {
		logger.Error("failed to parse jwt", "err", err)
		return nil, errInvalidToken
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !token.Valid || !ok {
		return nil, errInvalidToken
	}

	return claims, nil
}
