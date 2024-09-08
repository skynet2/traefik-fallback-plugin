package traefik_fallback_plugin

import (
	"bytes"
	"net/http"
)

type Writer struct {
	Headers    http.Header
	Buf        bytes.Buffer
	StatusCode int
}

func (w *Writer) Header() http.Header {
	if w.Headers == nil {
		w.Headers = make(http.Header)
	}

	return w.Headers
}

func (w *Writer) Write(bytes []byte) (int, error) {
	return w.Buf.Write(bytes)
}

func (w *Writer) WriteHeader(statusCode int) {
	w.StatusCode = statusCode
}
