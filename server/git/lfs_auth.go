package git

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/jwk"
	"github.com/charmbracelet/soft-serve/server/lfs"
	"github.com/charmbracelet/soft-serve/server/proto"
	"github.com/golang-jwt/jwt/v5"
)

// LFSAuthenticate implements teh Git LFS SSH authentication command.
// Context must have *config.Config, *log.Logger, proto.User.
// cmd.Args should have the repo path and operation as arguments.
func LFSAuthenticate(ctx context.Context, cmd ServiceCommand) error {
	if len(cmd.Args) < 2 {
		return errors.New("missing args")
	}

	logger := log.FromContext(ctx).WithPrefix("ssh.lfs-authenticate")
	operation := cmd.Args[1]
	if operation != lfs.OperationDownload && operation != lfs.OperationUpload {
		logger.Errorf("invalid operation: %s", operation)
		return errors.New("invalid operation")
	}

	user := proto.UserFromContext(ctx)
	if user == nil {
		logger.Errorf("missing user")
		return proto.ErrUserNotFound
	}

	repo := proto.RepositoryFromContext(ctx)
	if repo == nil {
		logger.Errorf("missing repository")
		return proto.ErrRepoNotFound
	}

	cfg := config.FromContext(ctx)
	kp, err := jwk.NewPair(cfg)
	if err != nil {
		logger.Error("failed to get JWK pair", "err", err)
		return err
	}

	now := time.Now()
	expiresIn := time.Minute * 5
	expiresAt := now.Add(expiresIn)
	claims := jwt.RegisteredClaims{
		Subject:   fmt.Sprintf("%s#%d", user.Username(), user.ID()),
		ExpiresAt: jwt.NewNumericDate(expiresAt), // expire in an hour
		NotBefore: jwt.NewNumericDate(now),
		IssuedAt:  jwt.NewNumericDate(now),
		Issuer:    cfg.HTTP.PublicURL,
		Audience: []string{
			repo.Name(),
		},
	}

	token := jwt.NewWithClaims(jwk.SigningMethod, claims)
	token.Header["kid"] = kp.JWK().KeyID
	j, err := token.SignedString(kp.PrivateKey())
	if err != nil {
		logger.Error("failed to sign token", "err", err)
		return err
	}

	href := fmt.Sprintf("%s/%s.git/info/lfs", cfg.HTTP.PublicURL, repo.Name())
	logger.Debug("generated token", "token", j, "href", href, "expires_at", expiresAt)

	return json.NewEncoder(cmd.Stdout).Encode(lfs.AuthenticateResponse{
		Header: map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", j),
		},
		Href:      href,
		ExpiresAt: expiresAt,
		ExpiresIn: expiresIn,
	})
}
