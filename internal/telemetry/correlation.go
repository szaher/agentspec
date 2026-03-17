package telemetry

import (
	"crypto/rand"
	"net/http"
	"time"

	"github.com/oklog/ulid/v2"
)

const correlationHeader = "X-Correlation-ID"

// CorrelationMiddleware extracts or generates a correlation ID for each request.
// If the incoming request has an X-Correlation-ID header, that value is used.
// Otherwise, a new ULID is generated. The correlation ID is injected into the
// request context and set on the response header.
func CorrelationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(correlationHeader)
		if id == "" {
			id = ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
		}

		ctx := WithCorrelationID(r.Context(), id)
		w.Header().Set(correlationHeader, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
