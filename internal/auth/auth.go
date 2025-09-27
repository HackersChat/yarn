package auth

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// Auther is an middleware interface used to restrict web handlers that either
// require authentication or already have authentication and redirect to the main
// handler.
type Auther interface {
	MustAuth(next httprouter.Handle) httprouter.Handle
}

// UserCreator is called by authentication implementations like the
// proxy auth that have their authentication handled by another system
// like a parent proxy or sso and require user accounts to be created
// on the fly. CreateUser therefore will take a username or unique-id and
// create the account if it doesn't already exist.
type UserCreator interface {
	CreateUser(user string, req *http.Request) error
}
