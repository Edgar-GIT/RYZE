package middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/api/auth"
)

// BodyLimit bounds the size of every request body. A body larger than maxBytes
// is rejected with 413 REQUEST_ENTITY_TOO_LARGE before any handler runs, and a
// request with no declared length is still read through a bounded reader so a
// streaming client cannot allocate unbounded memory.
//
// The body is buffered only up to the limit and then re-exposed unchanged, so
// handlers (including the payment webhooks, whose signature verification reads
// the raw bytes) observe the exact same content. Buffering in the middleware,
// rather than relying on http.MaxBytesReader alone, is what makes the rejection
// a deterministic 413: the JSON binding layer would otherwise report an
// oversized body as a generic 400.
//
// A maxBytes value that is not positive disables the limit; the server
// configuration always supplies a positive value.
func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body == nil || maxBytes <= 0 {
			c.Next()
			return
		}

		if c.Request.ContentLength > maxBytes {
			rejectOversizedBody(c)
			return
		}

		body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxBytes+1))
		if err != nil {
			auth.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Unable to read request body.", nil)
			c.Abort()
			return
		}

		if int64(len(body)) > maxBytes {
			rejectOversizedBody(c)
			return
		}

		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		c.Next()
	}
}

// rejectOversizedBody aborts the request with the project's error envelope and
// a 413 status.
func rejectOversizedBody(c *gin.Context) {
	auth.RespondError(c, http.StatusRequestEntityTooLarge, "REQUEST_ENTITY_TOO_LARGE", "Request body too large.", nil)
	c.Abort()
}
