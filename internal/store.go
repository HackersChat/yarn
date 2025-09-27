// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"go.mills.io/bitcask/v2"
	"go.mills.io/sessions"
)

var (
	ErrInvalidStore = errors.New("error: invalid store")
	ErrUserNotFound = errors.New("error: user not found")
	ErrFeedNotFound = errors.New("error: feed not found")
)

type Store interface {
	DB() *bitcask.Bitcask

	Backup(string) error
	Restore(string) error

	Merge() error
	Close() error
	Sync() error

	DelUser(username string) error
	HasUser(username string) bool
	GetUser(username string) (*User, error)
	SetUser(username string, user *User) error
	LenUsers() int64
	SearchUsers(prefix string) []string
	GetAllUsers() ([]*User, error)

	GetSession(sid string) (*sessions.Session, error)
	SetSession(sid string, sess *sessions.Session) error
	HasSession(sid string) bool
	DelSession(sid string) error
	SyncSession(sess *sessions.Session) error
	LenSessions() int64
	GetAllSessions() ([]*sessions.Session, error)
}

type StoreFactory func() (Store, error)

func retryableStore(newStore StoreFactory, maxRetries int, retryableErrors []error) (store Store, err error) {
retry:
	for i := 0; i < maxRetries; i++ {
		store, err = newStore()
		if err != nil {
			for n, retryableError := range retryableErrors {
				if errors.Is(err, retryableError) {
					log.WithError(err).Warnf(
						"retryable error %s [%d/%d] (retrying in 1s)",
						err, (n + 1), maxRetries,
					)
					time.Sleep(time.Second * 1)
					continue retry
				}
			}
			return
		}
		return
	}
	err = fmt.Errorf("error creating store (tried %d times, last error: %w)", maxRetries, err)
	return
}

func NewStore(store string) (Store, error) {
	u, err := ParseURI(store)
	if err != nil {
		return nil, fmt.Errorf("error parsing store uri: %s", err)
	}

	switch u.Type {
	case "bitcask":
		return retryableStore(
			func() (Store, error) { return newBitcaskStore(u.Path) },
			3, []error{&bitcask.ErrBadConfig{}, &bitcask.ErrBadMetadata{}},
		)
	default:
		return nil, ErrInvalidStore
	}
}
