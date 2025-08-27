package web

import (
	"fmt"
	"io"
	"net/http"
)

func renderStatus(code int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(code)
		io.WriteString(w, fmt.Sprintf("%d %s", code, http.StatusText(code))) //nolint:errcheck,gosec
	}
}
