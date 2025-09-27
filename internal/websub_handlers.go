package internal

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// WebSubHandler ...
func (s *Server) WebSubHandler() httprouter.Handle {
	isAdminUser := IsAdminUserFactory(s.config)

	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		r.Body = http.MaxBytesReader(w, r.Body, 1024)
		defer r.Body.Close()

		if r.Method == http.MethodPost {
			websub.WebSubEndpoint(w, r)
			return
		}

		if r.Method == http.MethodGet {
			ctx := NewContext(s, r)

			if !isAdminUser(ctx.User) {
				ctx.Error = true
				ctx.Message = "You are not a Pod Owner!"
				w.WriteHeader(http.StatusForbidden)
				s.render("403", r, w, ctx)
				return
			}

			websub.DebugEndpoint(w, r)
			return
		}

		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// NotifyHandler ...
func (s *Server) NotifyHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		r.Body = http.MaxBytesReader(w, r.Body, 1024)
		defer r.Body.Close()
		websub.NotifyEndpoint(w, r)
	}
}
