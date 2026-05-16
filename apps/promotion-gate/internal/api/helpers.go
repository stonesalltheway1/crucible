package api

import (
	"errors"
	"net/http"
)

// errEmptyBody is returned when a JSON body is missing (callers may want
// to treat as "use defaults" instead of erroring).
var errEmptyBody = errors.New("empty body")

// logMiddleware is a placeholder so the gate-internal api package mirrors
// the control-plane API's middleware shape. The bot of the request is
// already logged in ServeHTTP.
func logMiddleware(_ any, h http.Handler) http.Handler { return h }
