// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"time"

	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
	"go.mills.io/sessions"
)

// SessionStore ...
type SessionStore struct {
	store  Store
	cached *cache.Cache
}

func NewSessionStore(store Store, sessionCacheTTL time.Duration) sessions.Store {
	return &SessionStore{
		store:  store,
		cached: cache.New(sessionCacheTTL, time.Minute*5),
	}
}

func (s *SessionStore) LenSessions() int64 {
	return int64(s.cached.ItemCount()) + s.store.LenSessions()
}

func (s *SessionStore) GetSession(sid string) (*sessions.Session, error) {
	val, found := s.cached.Get(sid)
	if found {
		return val.(*sessions.Session), nil
	}

	return s.store.GetSession(sid)
}

func (s *SessionStore) SetSession(sid string, sess *sessions.Session) error {
	s.cached.Set(sid, sess, cache.DefaultExpiration)
	if persist, ok := sess.Get("persist"); !ok || persist != "1" {
		return nil
	}

	return s.store.SetSession(sid, sess)
}

func (s *SessionStore) HasSession(sid string) bool {
	_, ok := s.cached.Get(sid)
	if ok {
		return true
	}

	return s.store.HasSession(sid)
}

func (s *SessionStore) DelSession(sid string) error {
	if s.store.HasSession(sid) {
		if err := s.store.DelSession(sid); err != nil {
			log.WithError(err).Errorf("error deleting persistent session %s", sid)
			return err
		}
	}
	s.cached.Delete(sid)
	return nil
}

func (s *SessionStore) SyncSession(sess *sessions.Session) error {
	if persist, ok := sess.Get("persist"); ok && persist == "1" {
		if err := s.store.SetSession(sess.ID, sess); err != nil {
			log.WithError(err).Errorf("error persisting session %s", sess.ID)
			return err
		}
	}

	return s.SetSession(sess.ID, sess)
}

func (s *SessionStore) GetAllSessions() ([]*sessions.Session, error) {
	var cachedSessions []*sessions.Session
	for _, item := range s.cached.Items() {
		sess := item.Object.(*sessions.Session)
		cachedSessions = append(cachedSessions, sess)
	}
	persistedSessions, err := s.store.GetAllSessions()
	if err != nil {
		log.WithError(err).Error("error getting all persisted sessions")
		return cachedSessions, err
	}
	return append(cachedSessions, persistedSessions...), nil
}
