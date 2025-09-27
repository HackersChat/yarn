package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andyleap/microformats"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

// HApp ...
type HApp struct {
	Name string
	URL  *url.URL
	Logo *url.URL
}

func (h HApp) String() string {
	if h.Name == "" {
		return h.URL.Hostname()
	}

	return h.Name
}

// GetIndieClientInfo ...
func GetIndieClientInfo(conf *Config, clientID string) (h HApp, err error) {
	u, err := url.Parse(clientID)
	if err != nil {
		log.WithError(err).Errorf("error parsing  url: %s", clientID)
		return h, err
	}
	h.URL = u

	res, err := RequestHTTP(conf, "GET", clientID, nil)
	if err != nil {
		log.WithError(err).Errorf("error making client request to %s", clientID)
		return h, err
	}
	defer res.Body.Close()

	body, err := html.Parse(res.Body)
	if err != nil {
		log.WithError(err).Errorf("error parsing source %s", clientID)
		return h, err
	}

	p := microformats.New()
	data := p.ParseNode(body, u)

	h.URL = u

	getHApp := func(data *microformats.Data) (*microformats.MicroFormat, error) {
		if data != nil {
			for _, item := range data.Items {
				if HasString(item.Type, "h-app") {
					return item, nil
				}
			}
		}
		return nil, errors.New("error: no entry found")
	}

	happ, err := getHApp(data)
	if err != nil {
		return h, err
	}

	if names, ok := happ.Properties["name"]; ok && len(names) > 0 {
		if name, ok := names[0].(string); ok {
			h.Name = name
		}
	}

	if logos, ok := happ.Properties["logo"]; ok && len(logos) > 0 {
		if logo, ok := logos[0].(string); ok {
			if u, err := url.Parse(logo); err != nil {
				h.Logo = u
			} else {
				log.WithError(err).Warnf("error parsing logo %s", logo)
			}
		}
	}

	return h, nil
}

// ValidateIndieRedirectURL ...
func ValidateIndieRedirectURL(clientID, redirectURI string) error {
	u1, err := url.Parse(clientID)
	if err != nil {
		log.WithError(err).Errorf("error parsing  clientID: %s", clientID)
		return err
	}

	u2, err := url.Parse(clientID)
	if err != nil {
		log.WithError(err).Errorf("error parsing redirectURI: %s", redirectURI)
		return err
	}

	if u1.Scheme != u2.Scheme {
		return errors.New("invalid redirect url, mismatched scheme")
	}

	if u1.Hostname() != u2.Hostname() {
		return errors.New("invalid redirect url, mismatched hostname")
	}

	if u1.Port() != u2.Port() {
		return errors.New("invalid redirect url, mismatched port")
	}

	return nil
}

// IndieAuthHandler ...
func (s *Server) IndieAuthHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := NewContext(s, r)

		clientID := r.FormValue("client_id")
		redirectURI := r.FormValue("redirect_uri")
		state := r.FormValue("state")

		if clientID == "" || redirectURI == "" || state == "" {
			log.Warn("missing authentication parameters")

			if r.Method == http.MethodHead {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}

			s.renderError(r, w, http.StatusBadRequest,
				"Error one or more authentication parameters are missing")
			return
		}

		/* TODO: What is `response_type` used for?
		responseType := r.FormValue("response_type")
		if responseType == "" {
			responseType = "id"
		}
		responseType = strings.ToLower(responseType)
		*/

		happ, err := GetIndieClientInfo(s.config, clientID)
		if err != nil {
			log.WithError(err).Warnf("error retrieving client information from %s", clientID)
		}

		if err := ValidateIndieRedirectURL(clientID, redirectURI); err != nil {
			log.WithError(err).Errorf("error validating redirectURI %s from client %s", redirectURI, clientID)

			if r.Method == http.MethodHead {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}

			s.renderError(r, w, http.StatusBadRequest, "Error validating redirect url")
			return
		}

		ctx.PromptTitle = s.tr(ctx, "IndieAuthTitle")
		ctx.PromptMessage = fmt.Sprintf(
			s.tr(ctx, "IndieAuthMessage"),
			happ, s.config.Name,
		)
		ctx.PromptCallback = fmt.Sprintf(
			"/indieauth/callback?redirect_uri=%s&state=%s",
			redirectURI, state,
		)
		ctx.PromptApprove = s.tr(ctx, "IndieAuthApprove")
		ctx.PromptCancel = s.tr(ctx, "IndieAuthCancel")
		ctx.PromptTarget = ""

		s.render("prompt", r, w, ctx)
	}
}

// IndieAuthCallbackHandler ...
func (s *Server) IndieAuthCallbackHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := NewContext(s, r)

		redirectURI := r.FormValue("redirect_uri")
		state := r.FormValue("state")

		if redirectURI == "" || state == "" {
			log.Warn("missing callback parameters")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		u, err := url.Parse(redirectURI)
		if err != nil {
			log.WithError(err).Errorf("error parsing redirectURI %s", redirectURI)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// TOOD: Make the expiry time configurable?
		expiryTime := time.Now().Add(30 * time.Minute).Unix()

		// Create auth token
		token := jwt.NewWithClaims(
			jwt.SigningMethodHS256,
			jwt.MapClaims{
				"expiresAt": expiryTime,
				"username":  ctx.Username,
			},
		)
		tokenString, err := token.SignedString([]byte(s.config.MagicLinkSecret))
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		parts := strings.SplitN(tokenString, ".", 3)
		tokenCache.Inc(parts[2])

		v := url.Values{}
		v.Add("code", tokenString)
		v.Add("state", state)
		u.RawQuery = v.Encode()

		http.Redirect(w, r, u.String(), http.StatusFound)
	}
}

// IndieAuthVerifyHandler ...
func (s *Server) IndieAuthVerifyHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		log.Debugf("IndieAuthVerify() ...")

		clientID := r.FormValue("client_id")
		redirectURI := r.FormValue("redirect_uri")
		code := r.FormValue("code")

		if clientID == "" || redirectURI == "" || code == "" {
			log.WithField("clientID", clientID).
				WithField("redirect_uri", redirectURI).
				WithField("code", code).
				Warn("missing verification parameters")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		indieAuthToken := code

		// Check if token is valid
		token, err := jwt.Parse(indieAuthToken, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			return []byte(s.config.MagicLinkSecret), nil
		})
		if err != nil {
			log.WithError(err).Error("error validing indieauth token")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		if tokenCache.Get(token.Signature) == 0 {
			log.Warn("no valid indieauth token found")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		tokenCache.Dec(token.Signature)

		var username string

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			expiresAt := int(claims["expiresAt"].(float64))
			username = string(claims["username"].(string))

			now := time.Now()
			secs := now.Unix()

			// Check token expiry
			if secs > int64(expiresAt) {
				log.Warn("indieauth token expired")
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
		} else {
			log.Warn("invalid indieauth token or invalid claims")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		me := map[string]string{
			"me": UserURL(s.config.URLForUser(username)),
		}

		data, err := json.Marshal(me)
		if err != nil {
			log.WithError(err).Error("error serializing me response")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}
}
