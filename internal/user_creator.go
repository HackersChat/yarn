package internal

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"git.mills.io/yarnsocial/yarn/internal/auth"
)

type userCreator struct {
	config *Config
	db     Store
}

func NewUserCreator(config *Config, db Store) auth.UserCreator {
	return &userCreator{config: config, db: db}
}

// CreateUser automatically creates a user account for `user` if there is no user found
// by that username in the store `db`, otherwise it does nothing. If an error occurs when
// trying to check for a user or create a new user account, an error is returned.
func (uc *userCreator) CreateUser(username string, req *http.Request) error {
	if uc.db.HasUser(username) {
		return nil
	}

	p := filepath.Join(uc.config.Data, feedsDir)
	if err := os.MkdirAll(p, 0755); err != nil {
		return fmt.Errorf("error creating feeds directory %s: %w", p, err)
	}

	fn := filepath.Join(p, username)
	if _, err := os.Stat(fn); err == nil {
		return fmt.Errorf("feed for user %s already exists", username)
	}

	if err := os.WriteFile(fn, []byte{}, 0644); err != nil {
		return fmt.Errorf("error creating new user feed %s: %w", username, err)
	}

	user := NewUser()
	user.Username = username
	user.URL = URLForUser(uc.config.BaseURL, username)
	user.CreatedAt = time.Now()

	if err := uc.db.SetUser(username, user); err != nil {
		return fmt.Errorf("error saving user object for new user %s: %w", username, err)
	}

	return nil
}
