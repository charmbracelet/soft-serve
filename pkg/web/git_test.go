package web

import (
	"bytes"
	"errors"
	"io"
	"net/http/httptest"
	"testing"
)

// shortReader returns n bytes then io.EOF in the same Read call,
// simulating a reader that signals EOF alongside the last chunk.
type eofWithDataReader struct {
	data []byte
	pos  int
}

func (r *eofWithDataReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	if r.pos >= len(r.data) {
		return n, io.EOF // data AND EOF in the same call
	}
	return n, nil
}

// shortWriter wraps a ResponseWriter and truncates each Write to limit bytes.
type shortWriteResponseWriter struct {
	*httptest.ResponseRecorder
	limit int
}

func (w *shortWriteResponseWriter) Write(p []byte) (int, error) {
	if len(p) > w.limit {
		p = p[:w.limit]
	}
	return w.ResponseRecorder.Write(p)
}

// errorReader returns an error after some bytes.
type errorReader struct {
	data    []byte
	pos     int
	failAt  int
	failErr error
}

func (r *errorReader) Read(p []byte) (int, error) {
	if r.pos >= r.failAt {
		return 0, r.failErr
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func TestFlushResponseWriterReadFrom(t *testing.T) {
	t.Run("copies all bytes including last chunk returned with EOF", func(t *testing.T) {
		data := bytes.Repeat([]byte("x"), 2048) // > 1 buffer
		rec := httptest.NewRecorder()
		fw := &flushResponseWriter{rec}

		n, err := fw.ReadFrom(&eofWithDataReader{data: data})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if int(n) != len(data) {
			t.Errorf("reported %d bytes; want %d", n, len(data))
		}
		if !bytes.Equal(rec.Body.Bytes(), data) {
			t.Errorf("body mismatch: got %d bytes, want %d", rec.Body.Len(), len(data))
		}
	})

	t.Run("returns io.ErrShortWrite when writer truncates", func(t *testing.T) {
		data := []byte("hello world")
		rec := httptest.NewRecorder()
		sw := &shortWriteResponseWriter{ResponseRecorder: rec, limit: 5}
		fw := &flushResponseWriter{sw}

		_, err := fw.ReadFrom(bytes.NewReader(data))
		if !errors.Is(err, io.ErrShortWrite) {
			t.Errorf("expected io.ErrShortWrite; got %v", err)
		}
	})

	t.Run("propagates non-EOF read errors", func(t *testing.T) {
		sentinel := errors.New("read failed")
		rec := httptest.NewRecorder()
		fw := &flushResponseWriter{rec}

		_, err := fw.ReadFrom(&errorReader{
			data:    []byte("partial"),
			failAt:  3,
			failErr: sentinel,
		})
		if !errors.Is(err, sentinel) {
			t.Errorf("expected sentinel error; got %v", err)
		}
	})
}
