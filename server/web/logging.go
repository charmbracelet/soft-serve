package web

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/charmbracelet/log"
	"github.com/dustin/go-humanize"
)

// logWriter is a wrapper around http.ResponseWriter that allows us to capture
// the HTTP status code and bytes written to the response.
type logWriter struct {
	http.ResponseWriter
	code, bytes int
}

var _ http.ResponseWriter = (*logWriter)(nil)

var _ http.Flusher = (*logWriter)(nil)

var _ http.Hijacker = (*logWriter)(nil)

var _ http.CloseNotifier = (*logWriter)(nil)

// Write implements http.ResponseWriter.
func (r *logWriter) Write(p []byte) (int, error) {
	written, err := r.ResponseWriter.Write(p)
	r.bytes += written
	return written, err
}

// Note this is generally only called when sending an HTTP error, so it's
// important to set the `code` value to 200 as a default.
func (r *logWriter) WriteHeader(code int) {
	r.code = code
	r.ResponseWriter.WriteHeader(code)
}

// Flush implements http.Flusher.
func (r *logWriter) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// CloseNotify implements http.CloseNotifier.
func (r *logWriter) CloseNotify() <-chan bool {
	if cn, ok := r.ResponseWriter.(http.CloseNotifier); ok {
		return cn.CloseNotify()
	}
	return nil
}

// Hijack implements http.Hijacker.
func (r *logWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := r.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, fmt.Errorf("http.Hijacker not implemented")
}

// NewLoggingMiddleware returns a new logging middleware.
func NewLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := log.FromContext(r.Context())
		start := time.Now()
		writer := &logWriter{code: http.StatusOK, ResponseWriter: w}
		logger.Debug("request",
			"method", r.Method,
			"uri", r.RequestURI,
			"addr", r.RemoteAddr)
		next.ServeHTTP(writer, r)
		elapsed := time.Since(start)
		logger.Debug("response",
			"status", fmt.Sprintf("%d %s", writer.code, http.StatusText(writer.code)),
			"bytes", humanize.Bytes(uint64(writer.bytes)),
			"time", elapsed)
	})
}
