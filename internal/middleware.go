package internal

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// CORSMiddleware adds CORS headers to the response. It should be used as the first middleware in the chain.
func CORSMiddleware(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter,
		r *http.Request, ps httprouter.Params) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		next(w, r, ps)
	}
}

// PeerDetectionHandler wraps an http.Handler to perform peer detection on each incoming request.
func PeerDetectionHandler(peerDetector *PeerDetector, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Perform peer detection based on request headers.
		peerDetector.DetectFromRequest(r)

		// Continue with the next handler in the chain.
		next.ServeHTTP(w, r)
	})
}
