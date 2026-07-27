package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultErrorHandler(t *testing.T) {
	o := newServerOptions()

	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"api error: not found", ErrNotFound("nope"), http.StatusNotFound, `"message":"nope"`},
		{"api error: forbidden", ErrForbidden("denied"), http.StatusForbidden, `"message":"denied"`},
		{"wrapped api error", fmt.Errorf("ctx: %w", ErrBadRequest("bad")), http.StatusBadRequest, `"message":"bad"`},
		{"unknown error -> 500", errors.New("boom"), http.StatusInternalServerError, `"message":"internal server error"`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)

			o.errorHandler(rec, req, tc.err)

			assert.Equal(t, tc.wantCode, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
			assert.Contains(t, rec.Body.String(), tc.wantBody)
		})
	}
}

func TestErrorConstructors(t *testing.T) {
	assert.Equal(t, http.StatusBadRequest, ErrBadRequest("x").StatusCode())
	assert.Equal(t, http.StatusForbidden, ErrForbidden("x").StatusCode())
	assert.Equal(t, http.StatusNotFound, ErrNotFound("x").StatusCode())
	assert.Equal(t, http.StatusInternalServerError, ErrInternal("x").StatusCode())

	assert.Equal(t, "boom", ErrNotFound("boom").Error())
}

func TestErrorImplementsAPIError(t *testing.T) {
	var err error = ErrNotFound("x")

	var ae APIError
	assert.True(t, errors.As(err, &ae))
	assert.Equal(t, http.StatusNotFound, ae.StatusCode())
}
