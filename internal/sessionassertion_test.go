package internal

import (
	"testing"

	"github.com/andreadipersio/securecookie"
	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/assert"
	"go.mills.io/sessions"
)

// This file defines some assertion helpers for sessions in the end to end
// tests. It's inspired by httpexpect, but uses a more compact and direct
// call style.

func extractSID(t *testing.T, yarndTokenCookie *httpexpect.Cookie) string {
	sid, err := securecookie.DecodeSignedValue(srv.config.CookieSecret, "yarnd_token", yarndTokenCookie.Value().Raw())
	assert.NoError(t, err, "decoding signed yarnd_token cookie failed")
	assert.NotEmpty(t, sid, "extracted session ID from signed yarnd_token cookie is empty")
	return sid
}

func assertNoSession(t *testing.T, yarndTokenCookie *httpexpect.Cookie) {
	sid := extractSID(t, yarndTokenCookie)
	sess, err := srv.sc.GetSession(sid)
	assert.ErrorIsf(t, err, sessions.ErrSessionNotFound,
		"session '%s' should not be present in session store, but is: %#v", sid, sess)
}

func assertSession(t *testing.T, yarndTokenCookie *httpexpect.Cookie) *SessionAssertion {
	sid := extractSID(t, yarndTokenCookie)
	sess, err := srv.sc.GetSession(sid)
	assert.NoErrorf(t, err, "expected session '%s' to be present in session store", sid)
	return &SessionAssertion{t, sess}
}

type SessionAssertion struct {
	t    *testing.T
	sess *sessions.Session
}

func (sa *SessionAssertion) NoUsername() *SessionAssertion {
	assert.NotContains(sa.t, sa.sess.Data, "username", "username in session should not be present")
	return sa
}

func (sa *SessionAssertion) Username(expectedUsername string) *SessionAssertion {
	assert.Equal(sa.t, expectedUsername, sa.sess.Data["username"], "username in session does not match")
	return sa
}

func (sa *SessionAssertion) NoCaptchaText() *SessionAssertion {
	assert.NotContains(sa.t, sa.sess.Data, "captchaText", "captchaTest in session should not be present")
	return sa
}

func (sa *SessionAssertion) HasCaptchaText() *SessionAssertion {
	assert.NotEmpty(sa.t, sa.sess.Data["captchaText"], "captchaText in session should be present")
	return sa
}

func (sa *SessionAssertion) CaptchaText() string {
	return sa.HasCaptchaText().sess.Data["captchaText"]
}
