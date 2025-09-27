package internal

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"go.mills.io/webfinger"
)

// WebFingerHandler ...
func (s *Server) WebFingerHandler() httprouter.Handle {
	wr := NewWebFingerResolver(s.config, s.db)
	wf := webfinger.Default(wr)
	wf.NoTLSHandler = nil // Disabled as it is expected that most `yarnd` (Yarn.social pods) instances are run behind a reverse proxy with TLS

	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		r.Body = http.MaxBytesReader(w, r.Body, 1024)
		defer r.Body.Close()

		wf.ServeHTTP(w, r)
	}
}
