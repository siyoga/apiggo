package server

import (
	"bytes"
	"net/http"
)

// bufferedResponseWriter captures a handler's response in memory instead of
// streaming it to the client. This lets a panic (or error) mid-handler discard
// the partial output and replace it with a clean error response.
//
// Trade-off: the full body is held in memory and streaming/flushing is disabled
// (http.Flusher, SSE, chunked) — acceptable for the JSON APIs apiggo targets.
type bufferedResponseWriter struct {
	header      http.Header
	status      int
	body        bytes.Buffer
	wroteHeader bool
}

func newBufferedResponseWriter() *bufferedResponseWriter {
	return &bufferedResponseWriter{
		header: make(http.Header),
		status: http.StatusOK,
	}
}

func (b *bufferedResponseWriter) Header() http.Header { return b.header }

func (b *bufferedResponseWriter) WriteHeader(status int) {
	if b.wroteHeader {
		return
	}
	b.status = status
	b.wroteHeader = true
}

func (b *bufferedResponseWriter) Write(p []byte) (int, error) {
	if !b.wroteHeader {
		b.WriteHeader(http.StatusOK)
	}
	return b.body.Write(p)
}

// flush writes the captured headers, status and body to the real writer.
func (b *bufferedResponseWriter) flush(w http.ResponseWriter) {
	dst := w.Header()
	for key, values := range b.header {
		dst[key] = values
	}
	w.WriteHeader(b.status)
	_, _ = w.Write(b.body.Bytes())
}
