package web

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/keygen"
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/jwk"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/golang-jwt/jwt/v5"
	"github.com/matryer/is"
)

// oracleFixture wires a router with a private repository owned by "owner"
// and a token for "stranger", a registered user with no access to it.
type oracleFixture struct {
	ctx     context.Context
	owner   proto.User
	be      *backend.Backend
	handler http.Handler
	token   string
}

func newOracleFixture(t *testing.T) *oracleFixture {
	t.Helper()
	is := is.New(t)
	ctx, be, _ := newLFSTestContext(t)

	// Persist a host key so tokens minted here verify against the same key
	// the server uses.
	cfg := config.FromContext(ctx)
	cfg.SSH.KeyPath = filepath.Join(cfg.DataPath, "ssh", "host_key")
	_, err := keygen.New(cfg.SSH.KeyPath, keygen.WithKeyType(keygen.Ed25519), keygen.WithWrite())
	is.NoErr(err)

	owner, err := be.CreateUser(ctx, "owner", proto.UserOptions{})
	is.NoErr(err)
	_, err = be.CreateRepository(ctx, "secret", owner, proto.RepositoryOptions{Private: true})
	is.NoErr(err)

	stranger, err := be.CreateUser(ctx, "stranger", proto.UserOptions{})
	is.NoErr(err)
	token, err := be.CreateAccessToken(ctx, stranger, "t", time.Now().Add(time.Hour))
	is.NoErr(err)

	return &oracleFixture{ctx: ctx, owner: owner, be: be, handler: NewRouter(ctx), token: token}
}

// do performs a request and returns the recorded response. An empty auth
// header means an anonymous request.
func (f *oracleFixture) do(method, path, auth string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), method, path, nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, req)
	return w
}

// answer is the part of a response a caller can compare between two
// repository names: status, the credentials challenge, and the body.
type answer struct {
	code    int
	wwwAuth string
	body    string
}

func answerOf(w *httptest.ResponseRecorder) answer {
	return answer{
		code:    w.Code,
		wwwAuth: w.Header().Get("WWW-Authenticate"),
		body:    w.Body.String(),
	}
}

// assertSameAnswer requests the same path for the private repository and
// for a missing one and checks the caller cannot tell them apart.
func (f *oracleFixture) assertSameAnswer(t *testing.T, method, path, auth string, wantCode int) {
	t.Helper()
	is := is.New(t)
	private := answerOf(f.do(method, fmt.Sprintf(path, "secret"), auth))
	missing := answerOf(f.do(method, fmt.Sprintf(path, "ghost"), auth))
	is.Equal(private.code, wantCode)
	is.Equal(missing, private)
}

// TestReceivePackDoesNotRevealPrivateRepositories covers the push paths,
// which used to answer 401 for a private repository and 404 for a missing
// one to the same signed-in user.
func TestReceivePackDoesNotRevealPrivateRepositories(t *testing.T) {
	is := is.New(t)
	f := newOracleFixture(t)
	auth := "Token " + f.token

	// Routes that carry ?service=git-receive-pack but never create a
	// repository must answer the same for private and missing.
	for _, path := range []string{
		"/%s.git/HEAD?service=git-receive-pack",
		"/%s.git/objects/info/packs?service=git-receive-pack",
	} {
		f.assertSameAnswer(t, http.MethodGet, path, auth, http.StatusNotFound)
		f.assertSameAnswer(t, http.MethodGet, path, "", http.StatusUnauthorized)
	}

	// The push advertisement and RPC answer the same for an anonymous
	// caller, who cannot create a repository.
	f.assertSameAnswer(t, http.MethodGet, "/%s.git/info/refs?service=git-receive-pack", "", http.StatusUnauthorized)
	f.assertSameAnswer(t, http.MethodPost, "/%s.git/git-receive-pack", "", http.StatusUnauthorized)

	// Nothing above created a repository.
	_, err := f.be.Repository(f.ctx, "ghost")
	is.True(err != nil)

	// A signed-in user's push to a private repository they cannot read is
	// a 404, the same as every other unreadable repository, while a push
	// to a new name still creates it.
	is.Equal(f.do(http.MethodGet, "/secret.git/info/refs?service=git-receive-pack", auth).Code, http.StatusNotFound)
	is.Equal(f.do(http.MethodPost, "/secret.git/git-receive-pack", auth).Code, http.StatusNotFound)
	is.Equal(f.do(http.MethodGet, "/ghost.git/info/refs?service=git-receive-pack", auth).Code, http.StatusOK)
	_, err = f.be.Repository(f.ctx, "ghost")
	is.NoErr(err)
}

// TestReceivePackReaderIsForbidden verifies a caller who can read but not
// write is told so rather than being treated as unable to see the repo.
func TestReceivePackReaderIsForbidden(t *testing.T) {
	is := is.New(t)
	f := newOracleFixture(t)
	ownerCtx := proto.WithUserContext(f.ctx, f.owner)
	is.NoErr(f.be.AddCollaborator(ownerCtx, "secret", "stranger", access.ReadOnlyAccess))

	auth := "Token " + f.token
	is.Equal(f.do(http.MethodGet, "/secret.git/info/refs?service=git-receive-pack", auth).Code, http.StatusForbidden)
	is.Equal(f.do(http.MethodGet, "/secret.git/info/refs?service=git-upload-pack", auth).Code, http.StatusOK)

	// The service parameter does not turn a read into a push off the
	// advertisement.
	is.Equal(f.do(http.MethodGet, "/secret.git/HEAD?service=git-receive-pack", auth).Code, http.StatusOK)
	want := answerOf(f.do(http.MethodGet, "/secret.git/objects/info/whatever", auth))
	is.Equal(want.code, http.StatusNotFound)
	is.Equal(answerOf(f.do(http.MethodGet, "/secret.git/objects/info/whatever?service=git-receive-pack", auth)), want)

	// A GET whose path ends in the push service name is routed to the RPC
	// handler with a nested repository name. It used to be authorized as a
	// push first, which could create that nested repository; now the
	// method is refused before anything else, for any caller.
	for _, name := range []string{"secret", "ghost"} {
		for _, a := range []string{auth, ""} {
			got := answerOf(f.do(http.MethodGet, "/"+name+"/objects/info/git-receive-pack", a))
			is.Equal(got.code, http.StatusMethodNotAllowed)
		}
	}
	_, err := f.be.Repository(f.ctx, "secret/objects/info")
	is.True(err != nil)
	_, err = f.be.Repository(f.ctx, "ghost/objects/info")
	is.True(err != nil)
}

// mintJWT signs a token for user the way `soft jwt` does.
func mintJWT(ctx context.Context, t *testing.T, user proto.User, audience ...string) string {
	t.Helper()
	is := is.New(t)
	cfg := config.FromContext(ctx)
	kp, err := jwk.NewPair(cfg)
	is.NoErr(err)

	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   fmt.Sprintf("%s#%d", user.Username(), user.ID()),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		NotBefore: jwt.NewNumericDate(now),
		IssuedAt:  jwt.NewNumericDate(now),
		Issuer:    cfg.HTTP.PublicURL,
		Audience:  audience,
	}
	token := jwt.NewWithClaims(jwk.SigningMethod, claims)
	token.Header["kid"] = kp.JWK().KeyID
	signed, err := token.SignedString(kp.PrivateKey())
	is.NoErr(err)
	return signed
}

// TestBearerTokenDoesNotRevealPrivateRepositories covers JWT auth, which
// used to reject any token for a missing repository as bad credentials
// (403) before checking it, while a valid token for a private repository
// identified the user and answered 404.
func TestBearerTokenDoesNotRevealPrivateRepositories(t *testing.T) {
	is := is.New(t)
	f := newOracleFixture(t)
	stranger, err := f.be.User(f.ctx, "stranger")
	is.NoErr(err)

	const path = "/%s.git/info/refs?service=git-upload-pack"
	mint := func(audience ...string) string {
		return "Bearer " + mintJWT(f.ctx, t, stranger, audience...)
	}

	// A token whose audience names the requested repository identifies the
	// user whether or not the repository exists.
	for _, name := range []string{"secret", "ghost"} {
		is.Equal(answerOf(f.do(http.MethodGet, fmt.Sprintf(path, name), mint(name))).code, http.StatusNotFound)
	}
	private := answerOf(f.do(http.MethodGet, fmt.Sprintf(path, "secret"), mint("secret")))
	missing := answerOf(f.do(http.MethodGet, fmt.Sprintf(path, "ghost"), mint("ghost")))
	is.Equal(missing, private)

	// The same holds when the token names several repositories.
	private = answerOf(f.do(http.MethodGet, fmt.Sprintf(path, "secret"), mint("a", "secret", "b")))
	missing = answerOf(f.do(http.MethodGet, fmt.Sprintf(path, "ghost"), mint("a", "ghost", "b")))
	is.Equal(private.code, http.StatusNotFound)
	is.Equal(missing, private)

	// A token minted for some other repository, or for none, is bad
	// credentials either way.
	for _, audience := range [][]string{{"elsewhere"}, {"a", "b"}, {}} {
		f.assertSameAnswer(t, http.MethodGet, path, mint(audience...), http.StatusForbidden)
	}

	// A valid token still grants the user their own access.
	ownerCtx := proto.WithUserContext(f.ctx, f.owner)
	is.NoErr(f.be.AddCollaborator(ownerCtx, "secret", "stranger", access.ReadOnlyAccess))
	is.Equal(f.do(http.MethodGet, fmt.Sprintf(path, "secret"), mint("secret")).Code, http.StatusOK)

	// The audience is matched against the sanitized name, which is what
	// `soft jwt` mints, so a ".git" suffix in the token does not match.
	is.Equal(f.do(http.MethodGet, fmt.Sprintf(path, "secret"), mint("secret.git")).Code, http.StatusForbidden)
}
