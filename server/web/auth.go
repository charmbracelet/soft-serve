package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/proto"
	"github.com/golang-jwt/jwt/v5"
)

// authenticate authenticates the user from the request.
func authenticate(r *http.Request) (proto.User, error) {
	ctx := r.Context()
	logger := log.FromContext(ctx)

	// Check for auth header
	header := r.Header.Get("Authorization")
	if header != "" {
		logger.Debug("authorization", "header", header)

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 {
			return nil, errors.New("invalid authorization header")
		}

		// TODO: add basic, and token types
		be := backend.FromContext(ctx)
		switch strings.ToLower(parts[0]) {
		case "bearer":
			claims, err := getJWTClaims(ctx, parts[1])
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
			return nil, errors.New("invalid authorization header")
		}
	}

	logger.Debug("no authorization header")

	return nil, proto.ErrUserNotFound
}

// ErrInvalidToken is returned when a token is invalid.
var ErrInvalidToken = errors.New("invalid token")

func getJWTClaims(ctx context.Context, bearer string) (*jwt.RegisteredClaims, error) {
	cfg := config.FromContext(ctx)
	logger := log.FromContext(ctx).WithPrefix("http.auth")
	kp, err := cfg.SSH.KeyPair()
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
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !token.Valid || !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
