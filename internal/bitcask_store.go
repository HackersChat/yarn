// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"go.mills.io/bitcask/v2"
	"go.mills.io/sessions"
)

const (
	restoreFilename   = "db.restore.json"
	sessionsKeyPrefix = "/sessions"
	usersKeyPrefix    = "/users"
)

type kvPair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func exportKey(db *bitcask.Bitcask, w io.Writer) bitcask.KeyFunc {
	return func(key bitcask.Key) error {
		value, err := db.Get(key)
		if err != nil {
			return fmt.Errorf("error reading key %q: %w", key, err)
		}

		kv := kvPair{
			Key:   base64.StdEncoding.EncodeToString([]byte(key)),
			Value: base64.StdEncoding.EncodeToString(value),
		}

		data, err := json.Marshal(&kv)
		if err != nil {
			return fmt.Errorf("error serializing key %q: %w", key, err)
		}

		if n, err := w.Write(data); err != nil || n != len(data) {
			if err == nil && n != len(data) {
				err = fmt.Errorf("error not all data written %d/%d", n, len(data))
			}
			return fmt.Errorf("error exporting key %q: %w", key, err)
		}

		if _, err := w.Write([]byte("\n")); err != nil {
			return fmt.Errorf("error writing newline separator: %w", err)
		}

		return nil
	}
}

// BitcaskStore ...
type BitcaskStore struct {
	db *bitcask.Bitcask
}

func newBitcaskStore(path string) (*BitcaskStore, error) {
	db, err := bitcask.Open(
		path,

		// Session ID(s) are 88 bytes long: ceil(64 / 3) * 4
		// "/Sessions/" key prefix is 10 bytes long
		// TODO: Reduce this to 100 bytes?
		bitcask.WithMaxKeySize(256),

		// Project application database from accidental OOM kills
		bitcask.WithSyncWrites(true),
	)
	if err != nil {
		switch {
		case errors.Is(err, &bitcask.ErrBadConfig{}):
			log.WithError(err).Error("error opening database due to bad config")
			if osErr := os.Remove(filepath.Join(path, "config.json")); osErr != nil {
				log.WithError(osErr).Error("error removing bad config")
			}
		case errors.Is(err, &bitcask.ErrBadMetadata{}):
			log.WithError(err).Error("error opening database due to bad metadata")
			if osErr := os.Remove(filepath.Join(path, "meta.json")); osErr != nil {
				log.WithError(osErr).Error("error removing bad metadata")
			}
		}
		return nil, err
	}

	return &BitcaskStore{db: db}, nil
}

func (bs *BitcaskStore) scanKeys(prefix string) (keys []bitcask.Key, err error) {
	err = bs.db.Scan(bitcask.Key(prefix), func(key bitcask.Key) error {
		keys = append(keys, key)
		return nil
	})

	return
}

// DB ...
func (bs *BitcaskStore) DB() *bitcask.Bitcask {
	return bs.db
}

// Backup ...
func (bs *BitcaskStore) Backup(fn string) error {
	f, err := os.OpenFile(fn, os.O_CREATE|os.O_SYNC|os.O_TRUNC|os.O_WRONLY, os.FileMode(0600))
	if err != nil {
		return err
	}
	defer f.Close()

	return bs.DB().ForEach(exportKey(bs.DB(), f))
}

// Restore ...
func (bs *BitcaskStore) Restore(fn string) error {
	var kv kvPair

	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if err := json.Unmarshal(scanner.Bytes(), &kv); err != nil {
			return fmt.Errorf("error reading input: %w", err)
		}

		key, err := base64.StdEncoding.DecodeString(kv.Key)
		if err != nil {
			return fmt.Errorf("error decoding key %q: %w", kv.Key, err)
		}

		value, err := base64.StdEncoding.DecodeString(kv.Value)
		if err != nil {
			return fmt.Errorf("error decoding value for %q: %w", kv.Key, err)
		}

		if err := bs.DB().Put(key, value); err != nil {
			return fmt.Errorf("error writing key %q: %w", key, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	return nil
}

// Sync ...
func (bs *BitcaskStore) Sync() error {
	return bs.db.Sync()
}

// Close ...
func (bs *BitcaskStore) Close() error {
	log.Info("syncing store ...")
	if err := bs.db.Sync(); err != nil {
		log.WithError(err).Error("error syncing store")
		return err
	}

	log.Info("closing store ...")
	if err := bs.db.Close(); err != nil {
		log.WithError(err).Error("error closing store")
		return err
	}

	return nil
}

// Merge ...
func (bs *BitcaskStore) Merge() error {
	log.Info("merging store ...")
	if err := bs.db.Merge(); err != nil {
		log.WithError(err).Error("error merging store")
		return err
	}

	return nil
}

func (bs *BitcaskStore) HasUser(username string) bool {
	key := []byte(fmt.Sprintf("%s/%s", usersKeyPrefix, username))
	return bs.db.Has(key)
}

func (bs *BitcaskStore) DelUser(username string) error {
	key := []byte(fmt.Sprintf("%s/%s", usersKeyPrefix, username))
	return bs.db.Delete(key)
}

func (bs *BitcaskStore) GetUser(username string) (*User, error) {
	key := []byte(fmt.Sprintf("%s/%s", usersKeyPrefix, username))
	data, err := bs.db.Get(key)
	if err == bitcask.ErrKeyNotFound {
		return nil, ErrUserNotFound
	}
	return LoadUser(data)
}

func (bs *BitcaskStore) SetUser(username string, user *User) error {
	data, err := user.Bytes()
	if err != nil {
		return err
	}

	key := []byte(fmt.Sprintf("%s/%s", usersKeyPrefix, username))
	if err := bs.db.Put(key, data); err != nil {
		return err
	}
	return nil
}

func (bs *BitcaskStore) LenUsers() int64 {
	var count int64

	if err := bs.db.Scan(bitcask.Key(usersKeyPrefix), func(_ bitcask.Key) error {
		count++
		return nil
	}); err != nil {
		log.WithError(err).Error("error scanning")
	}

	return count
}

func (bs *BitcaskStore) SearchUsers(prefix string) []string {
	var keys []string

	if err := bs.db.Scan(bitcask.Key(usersKeyPrefix), func(key bitcask.Key) error {
		if strings.HasPrefix(strings.ToLower(string(key)), prefix) {
			keys = append(keys, strings.TrimPrefix(string(key), "/users/"))
		}
		return nil
	}); err != nil {
		log.WithError(err).Error("error scanning")
	}

	return keys
}

func (bs *BitcaskStore) GetAllUsers() ([]*User, error) {
	var users []*User

	keys, err := bs.scanKeys(usersKeyPrefix)
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		data, err := bs.db.Get(key)
		if err != nil {
			return nil, err
		}

		user, err := LoadUser(data)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (bs *BitcaskStore) GetSession(sid string) (*sessions.Session, error) {
	key := []byte(fmt.Sprintf("%s/%s", sessionsKeyPrefix, sid))
	data, err := bs.db.Get(key)
	if err != nil {
		if err == bitcask.ErrKeyNotFound {
			return nil, sessions.ErrSessionNotFound
		}
		return nil, err
	}
	sess := sessions.NewSession(bs)
	if err := sessions.LoadSession(data, sess); err != nil {
		return nil, err
	}
	return sess, nil
}

func (bs *BitcaskStore) SetSession(sid string, sess *sessions.Session) error {
	key := []byte(fmt.Sprintf("%s/%s", sessionsKeyPrefix, sid))

	data, err := sess.Bytes()
	if err != nil {
		return err
	}

	return bs.db.Put(key, data)
}

func (bs *BitcaskStore) HasSession(sid string) bool {
	key := []byte(fmt.Sprintf("%s/%s", sessionsKeyPrefix, sid))
	return bs.db.Has(key)
}

func (bs *BitcaskStore) DelSession(sid string) error {
	key := []byte(fmt.Sprintf("%s/%s", sessionsKeyPrefix, sid))
	return bs.db.Delete(key)
}

func (bs *BitcaskStore) SyncSession(sess *sessions.Session) error {
	// Only persist sessions with a logged in user associated with an account
	// This saves resources as we don't need to keep session keys around for
	// sessions we may never load from the store again.
	if sess.Has("username") {
		return bs.SetSession(sess.ID, sess)
	}
	return nil
}

// LenSessions returns the number of sessions in the store.
func (bs *BitcaskStore) LenSessions() int64 {
	var count int64

	if err := bs.db.Scan(bitcask.Key(sessionsKeyPrefix), func(_ bitcask.Key) error {
		count++
		return nil
	}); err != nil {
		log.WithError(err).Error("error scanning")
	}

	return count
}

func (bs *BitcaskStore) GetAllSessions() ([]*sessions.Session, error) {
	var res []*sessions.Session

	keys, err := bs.scanKeys(sessionsKeyPrefix)
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		data, err := bs.db.Get(key)
		if err != nil {
			return nil, err
		}

		sess := sessions.NewSession(bs)
		if err := sessions.LoadSession(data, sess); err != nil {
			return nil, err
		}
		res = append(res, sess)
	}

	return res, nil
}
