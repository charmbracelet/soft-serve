package web

import (
	"net/http"
	"strings"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/git"
)

// TestAuthorize pins the decision order for every kind of caller against
// every kind of route and repository state. The property that matters for
// the existence oracle is that, for a caller who cannot read, the verdict
// for a private repository (!readable, NoAccess) equals the verdict for a
// missing one (!readable, whatever level the caller gets for a missing
// name), so the level alone must not change the answer.
func TestAuthorize(t *testing.T) {
	type route struct {
		name    string
		service git.Service
		file    string
		method  string
	}
	isLFS := func(rt route) bool { return strings.HasPrefix(rt.file, "info/lfs") }
	routes := []route{
		{"fetch advert", git.UploadPackService, "info/refs", http.MethodGet},
		{"fetch rpc", git.UploadPackService, "git-upload-pack", http.MethodPost},
		{"push advert", git.ReceivePackService, "info/refs", http.MethodGet},
		{"push rpc", git.ReceivePackService, "git-receive-pack", http.MethodPost},
		{"dumb file", "", "HEAD", http.MethodGet},
		{"lfs batch", "", "info/lfs/objects/batch", http.MethodPost},
		{"lfs upload", "", "info/lfs/objects/basic/00", http.MethodPut},
		{"lfs lock", "", "info/lfs/locks", http.MethodPost},
	}

	type caller struct {
		name      string
		anonymous bool
		badCreds  bool
		// level the caller gets for a missing repository
		missing access.AccessLevel
	}
	callers := []caller{
		{"anonymous", true, false, access.ReadOnlyAccess},
		{"anonymous no-access", true, false, access.NoAccess},
		{"bad credentials", true, true, access.ReadOnlyAccess},
		{"signed-in", false, false, access.ReadWriteAccess},
	}

	// Missing and private must be indistinguishable for every caller who
	// cannot read the private repository.
	for _, c := range callers {
		for _, rt := range routes {
			private := authorize(rt.service, rt.file, rt.method, false, access.NoAccess, c.anonymous, c.badCreds)
			missing := authorize(rt.service, rt.file, rt.method, false, c.missing, c.anonymous, c.badCreds)
			want := denyUnreadable
			switch {
			case c.badCreds && !isLFS(rt):
				// LFS renders bad credentials from denyRead so the body
				// is JSON; the verdict stays denyUnreadable.
				want = forbid
			case rt.service == git.ReceivePackService && !c.anonymous && isPushEndpoint(rt.file):
				// The one inherent difference: a push to a new name by a
				// user who may create repositories creates it.
				if missing != createThenServe {
					t.Errorf("%s/%s: missing: got %v, want createThenServe", c.name, rt.name, missing)
				}
				if private != denyUnreadable {
					t.Errorf("%s/%s: private: got %v, want denyUnreadable", c.name, rt.name, private)
				}
				continue
			}
			if private != want || missing != want {
				t.Errorf("%s/%s: private=%v missing=%v, want both %v", c.name, rt.name, private, missing, want)
			}
		}
	}

	// Readers and writers of an existing repository.
	tests := []struct {
		name      string
		route     route
		level     access.AccessLevel
		anonymous bool
		want      verdict
	}{
		{"anon reader fetch", routes[0], access.ReadOnlyAccess, true, serve},
		{"anon reader push", routes[2], access.ReadOnlyAccess, true, unauthorized},
		{"anon reader push rpc", routes[3], access.ReadOnlyAccess, true, unauthorized},
		{"anon reader dumb", routes[4], access.ReadOnlyAccess, true, serve},
		{"anon reader lfs batch", routes[5], access.ReadOnlyAccess, true, serve},
		{"anon reader lfs upload", routes[6], access.ReadOnlyAccess, true, forbidLFSWrite},
		{"anon reader lfs lock", routes[7], access.ReadOnlyAccess, true, forbidLFSWrite},
		{"reader fetch", routes[0], access.ReadOnlyAccess, false, serve},
		{"reader push", routes[2], access.ReadOnlyAccess, false, forbid},
		{"reader push rpc", routes[3], access.ReadOnlyAccess, false, forbid},
		{"reader lfs upload", routes[6], access.ReadOnlyAccess, false, forbidLFSWrite},
		{"writer push", routes[2], access.ReadWriteAccess, false, serve},
		{"writer push rpc", routes[3], access.ReadWriteAccess, false, serve},
		{"writer lfs upload", routes[6], access.ReadWriteAccess, false, serve},
		{"writer lfs lock", routes[7], access.ReadWriteAccess, false, serve},
		{"anon writer push", routes[2], access.ReadWriteAccess, true, serve},
	}
	for _, tt := range tests {
		got := authorize(tt.route.service, tt.route.file, tt.route.method, true, tt.level, tt.anonymous, false)
		if got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}

	// Bad credentials on a readable repository are refused, and never
	// create anything even when the anonymous level would allow a push.
	for _, rt := range routes {
		if got := authorize(rt.service, rt.file, rt.method, true, access.ReadOnlyAccess, true, true); got != forbid {
			t.Errorf("bad creds on readable %s: got %v, want forbid", rt.name, got)
		}
		want := forbid
		if isLFS(rt) {
			want = denyUnreadable
		}
		if got := authorize(rt.service, rt.file, rt.method, false, access.ReadWriteAccess, true, true); got != want {
			t.Errorf("bad creds on missing %s: got %v, want %v", rt.name, got, want)
		}
	}

	// A push may only create from the push endpoints.
	if got := authorize(git.ReceivePackService, "HEAD", http.MethodGet, false, access.ReadWriteAccess, false, false); got != denyUnreadable {
		t.Errorf("push service on HEAD: got %v, want denyUnreadable", got)
	}
}
