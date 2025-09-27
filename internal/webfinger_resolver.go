package internal

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"go.mills.io/webfinger"
	"go.yarn.social/types"
)

type webfingerResolver struct {
	conf *Config
	db   Store
}

func NewWebFingerResolver(conf *Config, db Store) webfinger.Resolver {
	return &webfingerResolver{conf, db}
}

// FindUser finds the user given the username and hostname.
func (wr *webfingerResolver) FindUser(username string, hostname, requestHost string, r []webfinger.Rel) (*webfinger.Resource, error) {
	if hostname != wr.conf.baseURL.Host {
		return nil, ErrFeedNotFound
	}

	nick := NormalizeUsername(username)
	if nick == "" {
		return nil, ErrFeedNotFound
	}

	var profile types.Profile

	if user, err := wr.db.GetUser(nick); err == nil {
		profile = user.Profile(wr.conf.BaseURL, nil)
	} else {
		log.WithError(err).Warnf("unable to load user or feed profile for %s", nick)
		return nil, fmt.Errorf("error loading feed %s: %w", nick, err)
	}

	res := webfinger.Resource{
		Subject: fmt.Sprintf("acct:%s@%s", nick, hostname),
		Aliases: []string{
			profile.URI,
			profile.Avatar,
			UserURL(profile.URI),
		},
		Links: []webfinger.Link{
			{
				HRef: profile.URI,
				Type: "text/plain",
				Rel:  webfinger.RelSelf,
			},
			{
				HRef: UserURL(profile.URI),
				Type: "text/html",
				Rel:  webfinger.RelProfilePage,
			},
			{
				HRef: profile.Avatar,
				Type: "image/png",
				Rel:  webfinger.RelAvatar,
			},
		},
	}

	return &res, nil
}

// DummyUser allows us to return a dummy user to avoid user-enumeration via webfinger 404s. This
// can be done in the webfinger code itself but then it would be obvious which users are real
// and which are not real via differences in how the implementation works vs how
// the general webfinger code works. This does not match the webfinger specification
// but is an extra precaution. Returning a NotFound error here will
// keep the webfinger 404 behavior.
func (wr *webfingerResolver) DummyUser(username string, hostname string, r []webfinger.Rel) (*webfinger.Resource, error) {
	return nil, ErrFeedNotFound
}

// IsNotFoundError returns true if the given error is a not found error.
func (wr *webfingerResolver) IsNotFoundError(err error) bool {
	return err == ErrFeedNotFound
}
