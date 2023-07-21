package web

import "net/http"

func renderStatus(code int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(code)
		w.Write([]byte(http.StatusText(code)))
	}
}
