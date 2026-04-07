package web

import (
	"bytes"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	gitb "github.com/charmbracelet/soft-serve/git"
	"github.com/gorilla/mux"
)

// maxRawBlobSize is the largest blob that getRawBlob will read into memory.
// Requests for blobs exceeding this size receive HTTP 413.
const maxRawBlobSize = 32 * 1024 * 1024 // 32 MiB

// getRawBlob serves the raw content of a single file at a given ref and path.
// It is registered as GET /{repo}/raw/{ref}/{filepath}.
//
// Access control is enforced by the withAccess middleware that wraps this
// handler: unauthenticated users see 401, insufficient-access users see 404
// (to avoid leaking repo existence). The handler never re-checks access.
//
// The Accept header controls delivery:
//   - "application/octet-stream" → Content-Disposition: attachment (download)
//   - anything else              → Content-Type inferred from extension or
//     binary detection (text/plain for text, application/octet-stream for binary)
//
// Note: dir is constructed and sanitised by the withParams middleware and must
// not be derived locally inside this handler.
func getRawBlob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dir := vars["dir"]
	ref := vars["ref"]
	filePath := vars["filepath"]

	if filePath == "" {
		renderBadRequest(w, r)
		return
	}

	repo, err := gitb.Open(dir)
	if err != nil {
		renderNotFound(w, r)
		return
	}

	// Resolve HEAD when no ref is given.
	if ref == "" || ref == "HEAD" {
		head, err := repo.HEAD()
		if err != nil {
			renderNotFound(w, r)
			return
		}
		ref = head.ID
	}

	tree, err := repo.LsTree(ref)
	if err != nil {
		renderNotFound(w, r)
		return
	}

	te, err := tree.TreeEntry(filePath)
	if err != nil {
		renderNotFound(w, r)
		return
	}

	// Must be a blob (file), not a tree (directory).
	if te.Type() != "blob" {
		renderNotFound(w, r)
		return
	}

	// Guard against OOM/DoS from very large blobs.
	// te.Size() calls `git cat-file -s` and silently returns 0 on error,
	// so this is a fast early-out only; the post-load check below is the
	// authoritative guard.
	if te.Size() > maxRawBlobSize {
		renderStatus(http.StatusRequestEntityTooLarge)(w, r)
		return
	}

	bts, err := te.Contents()
	if err != nil {
		renderInternalServerError(w, r)
		return
	}

	// Belt-and-suspenders: re-check after loading in case te.Size() returned
	// 0 due to a silent git subprocess error.
	if int64(len(bts)) > maxRawBlobSize {
		renderStatus(http.StatusRequestEntityTooLarge)(w, r)
		return
	}

	// Determine Content-Type from extension first, then fall back to binary
	// detection using the bytes already in memory (avoids a second git subprocess).
	contentType := mime.TypeByExtension(filepath.Ext(filePath))
	if contentType == "" {
		isBin, _ := gitb.IsBinary(bytes.NewReader(bts))
		if isBin {
			contentType = "application/octet-stream"
		} else {
			contentType = "text/plain; charset=utf-8"
		}
	}

	// Sanitise: downgrade any MIME type that a browser will execute scripts from.
	// This prevents stored-XSS when an attacker pushes an .html/.svg/.js file.
	contentType = sanitizeMIME(contentType)

	// X-Content-Type-Options prevents browsers from sniffing and upgrading the type.
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// If the client explicitly requests a binary stream, serve as download.
	// NOTE: sanitizeMIME has already run above. Any content-type set in this
	// block must itself be safe to serve without further sanitisation.
	if r.Header.Get("Accept") == "application/octet-stream" {
		contentType = "application/octet-stream"
		// Build a safe filename for Content-Disposition per RFC 6266 §4.3 and
		// RFC 5987: strip ASCII control characters to prevent header injection,
		// then emit both filename= (ASCII fallback) and filename*= (RFC 5987)
		// forms so all clients get the correct name.
		rawName := filepath.Base(filePath)
		safeName := strings.Map(func(r rune) rune {
			if r < 0x20 || r == 0x7F {
				return -1 // drop control character
			}
			return r
		}, rawName)
		asciiName := strings.ReplaceAll(safeName, `"`, `\"`)
		// Check whether the name is pure ASCII (no code point above 0x7E).
		isASCII := true
		for _, r := range safeName {
			if r > 0x7E {
				isASCII = false
				break
			}
		}
		if isASCII {
			w.Header().Set("Content-Disposition", `attachment; filename="`+asciiName+`"`)
		} else {
			// RFC 5987 encoded filename: UTF-8''<percent-encoded>.
			// url.PathEscape percent-encodes all non-unreserved characters except '/'.
			// Replace any literal '/' that might appear in a segment (shouldn't
			// after filepath.Base, but be safe).
			encoded := strings.ReplaceAll(url.PathEscape(safeName), "/", "%2F")
			w.Header().Set("Content-Disposition",
				`attachment; filename="`+asciiName+`"; filename*=UTF-8''`+encoded)
		}
	}

	// Mutable refs (branch names, tags) must not be cached by proxies.
	// A future improvement could set max-age for immutable SHA refs.
	w.Header().Set("Cache-Control", "no-store")

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(bts)), 10))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(bts)
}

// sanitizeMIME uses an allowlist to ensure only MIME types that cannot execute
// scripts in a browser are forwarded. Everything else is downgraded to
// text/plain to prevent stored-XSS from pushed .html/.svg/.js/.css/etc. files.
//
// Allowlisted categories:
//   - text/plain, application/octet-stream (safe by definition)
//   - application/json (data only, not rendered/executed by browsers)
//   - image/* except SVG (SVG allows embedded scripts)
//   - audio/*, video/* (media; cannot execute scripts)
//
// All other types — including text/html, text/css, *+xml, */javascript,
// application/pdf, font/*, multipart/* — are downgraded.
func sanitizeMIME(ct string) string {
	// Strip parameters for comparison (e.g. "text/html; charset=utf-8" → "text/html").
	base := ct
	if i := strings.Index(ct, ";"); i != -1 {
		base = strings.TrimSpace(ct[:i])
	}
	base = strings.ToLower(base)

	switch {
	case base == "text/plain",
		base == "application/octet-stream",
		base == "application/json":
		return ct
	case strings.HasPrefix(base, "image/") && base != "image/svg+xml":
		// SVG is excluded: it supports embedded <script> elements.
		return ct
	case strings.HasPrefix(base, "audio/"),
		strings.HasPrefix(base, "video/"):
		return ct
	}

	// Downgrade everything else (HTML, CSS, JavaScript, SVG, XML, PDF, fonts, …).
	return "text/plain; charset=utf-8"
}
