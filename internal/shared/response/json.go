package response

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	domainErr "github.com/gurkanfikretgunak/masterfabric-go/internal/shared/errors"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 512))
	},
}

// JSON writes a JSON response with the given status code and payload using buffer pooling.
func JSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if payload == nil {
		w.WriteHeader(status)
		return
	}

	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufferPool.Put(buf)

	if err := json.NewEncoder(buf).Encode(payload); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

// Error writes a JSON error response, mapping domain errors to HTTP codes.
func Error(w http.ResponseWriter, err error) {
	code := domainErr.HTTPStatusCode(err)
	msg := err.Error()
	if code >= http.StatusInternalServerError {
		slog.Error("request failed", "code", code, "error", err)
		msg = "an internal error occurred"
	}
	JSON(w, code, domainErr.ErrorResponse{
		Error:   http.StatusText(code),
		Message: msg,
		Code:    code,
	})
}

// Created writes a 201 Created JSON response.
func Created(w http.ResponseWriter, payload interface{}) {
	JSON(w, http.StatusCreated, payload)
}

// NoContent writes a 204 No Content response.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
