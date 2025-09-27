// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package indieweb

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	sync "github.com/sasha-s/go-deadlock"
	log "github.com/sirupsen/logrus"
	"willnorris.com/go/microformats"
)

const (
	defaultWebSubRedeliveryAttempts = 6
	defaultWebSubLeaseTime          = time.Hour
	defaultWebSubQueueSize          = 100
	defaulWebSubTimeout             = 5 * time.Minute
)

var (
	_ json.Marshaler   = (*Subscription)(nil)
	_ json.Unmarshaler = (*Subscription)(nil)
)

func fileExists(fn string) bool {
	if _, err := os.Stat(fn); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func generateRandomChallengeString() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

type callback struct {
	topic string
}

type notification struct {
	topic    string
	target   string
	attempts int
}

type verification struct {
	target       string
	topic        string
	callback     string
	challenge    string
	leaseSeconds int
	attempts     int
}

type Subscriber struct {
	Topic     string    `json:"topic"`
	Callback  string    `json:"callback"`
	Verified  bool      `json:"verified"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewSubscriber(topic, callback string) *Subscriber {
	return &Subscriber{
		Topic:     topic,
		Callback:  callback,
		CreatedAt: time.Now(),
	}
}

func (s *Subscriber) Expired() bool {
	return time.Now().After(s.ExpiresAt)
}

type Subscribers []*Subscriber

func (s Subscribers) Len() int           { return len(s) }
func (s Subscribers) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s Subscribers) Less(i, j int) bool { return s[i].ExpiresAt.Before(s[j].ExpiresAt) }

type Subscription struct {
	sync.RWMutex

	confirmed bool
	createdAt time.Time
	expiresAt time.Time

	Topic string
}

func NewSubscription(topic string) *Subscription {
	return &Subscription{
		Topic:     topic,
		createdAt: time.Now(),
	}
}

func (s *Subscription) MarshalJSON() ([]byte, error) {
	s.RLock()
	defer s.RUnlock()

	o := struct {
		Confirmed bool      `json:"confirmed"`
		ExpiresAt time.Time `json:"expires_at"`
		Topic     string    `json:"topic"`
	}{
		Confirmed: s.confirmed,
		ExpiresAt: s.expiresAt,
		Topic:     s.Topic,
	}
	return json.Marshal(o)
}

func (s *Subscription) UnmarshalJSON(data []byte) error {
	o := struct {
		Confirmed bool      `json:"confirmed"`
		ExpiresAt time.Time `json:"expires_at"`
		Topic     string    `json:"topic"`
	}{}

	if err := json.Unmarshal(data, &o); err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	s.confirmed = o.Confirmed
	s.expiresAt = o.ExpiresAt
	s.Topic = o.Topic

	return nil
}

func (s *Subscription) Confirmed() bool {
	s.RLock()
	defer s.RUnlock()

	return s.confirmed
}

func (s *Subscription) Confirm(leaseSeconds int) {
	s.Lock()
	s.confirmed = true
	s.expiresAt = time.Now().Add(time.Duration(leaseSeconds) * time.Second)
	s.Unlock()
}

func (s *Subscription) Timedout(timeout time.Duration) bool {
	s.RLock()
	defer s.RUnlock()

	return s.createdAt.Add(timeout).After(time.Now())
}

func (s *Subscription) Expired() bool {
	s.RLock()
	defer s.RUnlock()

	return time.Now().After(s.expiresAt)
}

type WebSubStats struct {
	Topics int

	Subscribers int
	Verified    int

	Subscriptions int
	Confirmed     int
}

type WebSub struct {
	sync.RWMutex

	fn       string
	endpoint string

	// WebSub Subscribers to this Hub
	subscribers map[string]Subscribers

	// WebSub Subscriptions from this client
	subscriptions map[string]*Subscription

	// inxbox for processing cwllbacks asynchronously for notifications for subscriptions from this client
	inbox       chan *callback
	inboxTicker *time.Ticker

	// outbox queue for sending notifications for subscribers to this hub
	outbox       chan *notification
	outboxTicker *time.Ticker

	// verify queue for sending verification requests to callbacks for new subscribers to this hub
	verify       chan *verification
	verifyTicker *time.Ticker

	cleanupTicker *time.Ticker
	stateTicker   *time.Ticker

	// Notify is the callback called when processing inbound notifications requests
	// from a hub we're subscribed to as a client for a given topic
	Notify func(topic string) error

	// ValidateTopic is a function that takes a `topic` string as input and returns `true` if it is
	// a valid topic or `false` otherwise. Consumers should override this field with a custom
	// topic validation function that suits the application.
	ValidateTopic func(topic string) bool
}

func NewWebSub(fn, endpoint string) *WebSub {
	ws := &WebSub{
		fn:            fn,
		endpoint:      endpoint,
		subscriptions: make(map[string]*Subscription),
		subscribers:   make(map[string]Subscribers),
		inbox:         make(chan *callback, defaultWebSubQueueSize),
		outbox:        make(chan *notification, defaultWebSubQueueSize),
		verify:        make(chan *verification, defaultWebSubQueueSize),

		Notify:        func(topic string) error { return nil },
		ValidateTopic: func(topic string) bool { return true },
	}

	ws.inboxTicker = time.NewTicker(1 * time.Second)
	go func() {
		for range ws.inboxTicker.C {
			ws.processInbox()
		}
	}()

	ws.outboxTicker = time.NewTicker(2 * time.Second)
	go func() {
		for range ws.outboxTicker.C {
			ws.processOutbox()
		}
	}()

	ws.verifyTicker = time.NewTicker(1 * time.Second)
	go func() {
		for range ws.verifyTicker.C {
			ws.processVerify()
		}
	}()

	ws.cleanupTicker = time.NewTicker(5 * time.Minute)
	go func() {
		for range ws.cleanupTicker.C {
			ws.cleanup()
		}
	}()

	ws.stateTicker = time.NewTicker(1 * time.Minute)
	go func() {
		for range ws.stateTicker.C {
			ws.Save()
		}
	}()

	return ws
}

func (ws *WebSub) Load() error {
	if !fileExists(ws.fn) {
		return nil
	}

	data, err := os.ReadFile(ws.fn)
	if err != nil {
		os.Remove(ws.fn)
		return fmt.Errorf("error loading state: %w", err)
	}

	state := struct {
		Subscribers map[string]Subscribers
	}{}

	if err := json.Unmarshal(data, &state); err != nil {
		os.Remove(ws.fn)
		return fmt.Errorf("error deserializing state: %w", err)
	}

	ws.Lock()
	ws.subscribers = state.Subscribers
	ws.Unlock()

	return nil
}

func (ws *WebSub) Save() error {
	ws.RLock()
	defer ws.RUnlock()

	state := struct {
		Subscribers map[string]Subscribers
	}{
		Subscribers: ws.subscribers,
	}

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("error serializing state: %s", err)
	}

	if err := os.WriteFile(ws.fn, data, 0644); err != nil {
		return fmt.Errorf("error saving state: %w", err)
	}

	return nil
}

func (ws *WebSub) cleanup() {
	ws.Lock()
	defer ws.Unlock()

	for topic, subscription := range ws.subscriptions {
		if subscription.Confirmed() {
			if subscription.Expired() {
				delete(ws.subscriptions, topic)
			}
		} else {
			if subscription.Timedout(defaulWebSubTimeout) {
				delete(ws.subscriptions, topic)
			}
		}
	}

	for topic, subscribers := range ws.subscribers {
		var newSubscribers Subscribers
		for _, subscriber := range subscribers {
			if !subscriber.Verified || (subscriber.Verified && !subscriber.Expired()) {
				newSubscribers = append(newSubscribers, subscriber)
			}
		}
		ws.subscribers[topic] = newSubscribers
	}
}

func (ws *WebSub) addSubscriber(subscriber *Subscriber) {
	ws.subscribers[subscriber.Topic] = append(ws.subscribers[subscriber.Topic], subscriber)
}

func (ws *WebSub) AddSubscriber(subscriber *Subscriber) {
	ws.Lock()
	defer ws.Unlock()

	ws.addSubscriber(subscriber)
}

func (ws *WebSub) getSubscriberFor(topic, callback string) (*Subscriber, int) {
	subscribers, ok := ws.subscribers[topic]
	if !ok {
		return nil, -1
	}

	for idx, subscriber := range subscribers {
		if subscriber.Callback == callback {
			return subscriber, idx
		}
	}

	return nil, -1
}

func (ws *WebSub) GetSubscriberFor(topic, callback string) (*Subscriber, int) {
	ws.RLock()
	defer ws.RUnlock()

	return ws.getSubscriberFor(topic, callback)
}

func (ws *WebSub) HasSubscriberFor(topic, callback string) bool {
	_, idx := ws.GetSubscriberFor(topic, callback)
	return idx != -1
}

func (ws *WebSub) delSubscriber(topic string, idx int) {
	subscribers := ws.subscribers[topic]
	ws.subscribers[topic] = append(subscribers[:idx], subscribers[idx+1:]...)
}

func (ws *WebSub) DelSubscriber(topic string, idx int) {
	ws.Lock()
	defer ws.Unlock()

	ws.delSubscriber(topic, idx)
}

func (ws *WebSub) Subscribe(uri, callback string) error {
	log.Debugf("creating websub subscription for %s", uri)

	u, err := url.Parse(uri)
	if err != nil {
		log.WithError(err).Errorf("error parsing uri %s", uri)
		return err
	}

	if _, err := url.Parse(callback); err != nil {
		log.WithError(err).Errorf("error parsing cwllback %s", callback)
		return err
	}

	hubEndpoint, selfURL, err := ws.GetHubEndpoint(u)
	if err != nil {
		log.WithError(err).Errorf("error discovering hub endpoint for %s", uri)
		return err
	}
	log.Debugf("found hub endpoint: %s", hubEndpoint.String())
	log.Debugf("found self url: %s", selfURL.String())

	topic := selfURL.String()

	ws.Lock()
	ws.subscriptions[topic] = NewSubscription(topic)
	ws.Unlock()

	values := make(url.Values)
	values.Set("hub.mode", "subscribe")
	values.Set("hub.topic", topic)
	values.Set("hub.callback", callback)
	log.Debugf("Sending websub subscription request to %s", hubEndpoint.String())
	log.Debugf("values: %q", values)
	res, err := http.PostForm(hubEndpoint.String(), values)
	if err != nil {
		log.WithError(err).Errorf(
			"error sending websub subscription request to hubEndpoint=%s",
			hubEndpoint.String(),
		)
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusAccepted {
		err := fmt.Errorf(
			"bad response %s from subscription request for to hubEndpoint=%s",
			res.Status, hubEndpoint.String(),
		)
		log.Error(err)
		return err
	}
	log.Debugf("successfully sent websub subscription to %s for %s with callback %s", hubEndpoint.String(), uri, callback)

	return nil
}

func (ws *WebSub) GetSubscription(topic string) *Subscription {
	ws.RLock()
	defer ws.RUnlock()

	return ws.subscriptions[topic]
}

func (ws *WebSub) IsSubscribed(topic string) bool {
	sub := ws.GetSubscription(topic)
	if sub == nil {
		return false
	}

	return sub.Confirmed() && !sub.Expired()
}

func (ws *WebSub) GetHubEndpoint(target *url.URL) (hubEndpoint *url.URL, selfURL *url.URL, err error) {
	res, err := http.Get(target.String())
	if err != nil {
		log.WithError(err).Error("error getting hub endpoint")
		return nil, nil, err
	}
	defer res.Body.Close()

	links := GetHeaderLinks(res.Header["Link"])
	log.Debugf("links: %v", links)
	for _, link := range links {
		for _, rel := range link.Params["rel"] {
			if rel == "hub" {
				hubEndpoint = link.URL
			}
			if rel == "self" {
				selfURL = link.URL
			}
		}
	}
	if hubEndpoint == nil {
		log.Debugf("no hub endpoint found in HTTP Header Links")
	}
	if selfURL == nil {
		log.Debugf("no self url found in HTTP Header Links")
	}
	if hubEndpoint != nil && selfURL != nil {
		return
	}

	/*
		DEBU[0007] links: [0xc000790080 0xc0007900c0 0xc0007901d0]
		DEBU[0007] no rel=hub link found for /user/stats/twtxt.txt
	*/

	data := microformats.Parse(res.Body, target)

	log.Debugf("Rels: %v", data.Rels["hub"])
	for _, link := range data.Rels["hub"] {
		u, err := url.Parse(link)
		if err != nil {
			log.WithError(err).Warn("error parsing hub link")
			continue
		}
		hubEndpoint = u
	}

	log.Debugf("Rels: %v", data.Rels["self"])
	for _, link := range data.Rels["self"] {
		u, err := url.Parse(link)
		if err != nil {
			log.WithError(err).Warn("error parsing self link")
			continue
		}
		selfURL = u
	}

	if hubEndpoint == nil {
		log.Debugf("no hub endpoint found in Document")
	}
	if selfURL == nil {
		log.Debugf("no self url found in Document")
	}
	if hubEndpoint != nil && selfURL != nil {
		return
	}

	return nil, nil, fmt.Errorf("no hub endpoint found")
}

func (ws *WebSub) SendNotification(topic string) {
	ws.RLock()
	defer ws.RUnlock()

	subs, ok := ws.subscribers[topic]
	if !ok {
		log.Debugf("no subscriptions found for %s", topic)
		return
	}
	log.Debugf("%d subscriptions found for %s", len(subs), topic)

	for _, sub := range subs {
		ws.outbox <- &notification{topic: topic, target: sub.Callback}
	}
}

func (ws *WebSub) NotifyEndpoint(w http.ResponseWriter, r *http.Request) {
	log.Debugf("NotifyEndpoint:")

	mode := r.FormValue("hub.mode")
	topic := r.FormValue("hub.topic")
	challenge := r.FormValue("hub.challenge")
	leaseSeconds := r.FormValue("hub.lease_seconds")

	log.Debugf("mode: %s", mode)
	log.Debugf("topic: %s", topic)
	log.Debugf("challenge: %s", challenge)

	if strings.ToLower(mode) == "subscribe" {
		if !ws.IsSubscribed(topic) {
			n, err := strconv.Atoi(leaseSeconds)
			if err != nil {
				log.WithError(err).Errorf("error parsing leaseSeconds %s", leaseSeconds)
				http.Error(w, "Bad hub.lease_seconds", http.StatusNotFound)
				return
			}

			sub := ws.GetSubscription(topic)
			if sub == nil {
				log.Debugf("no subscription found for topic=%s", topic)
				http.Error(w, "Subscription Not Found", http.StatusNotFound)
				return
			}
			sub.Confirm(n)
		}
		http.Error(w, challenge, http.StatusAccepted)
		return
	}

	var (
		hubEndpoint *url.URL
		selfURL     *url.URL
	)

	links := GetHeaderLinks(r.Header["Link"])
	log.Debugf("links: %v", links)
	for _, link := range links {
		for _, rel := range link.Params["rel"] {
			if rel == "hub" {
				hubEndpoint = link.URL
			}
			if rel == "self" {
				selfURL = link.URL
			}
		}
	}

	if hubEndpoint == nil {
		log.Debugf("hub endpoint not found in request Link headers")
		http.Error(w, "Missing rel=hub Link", http.StatusNotFound)
		return
	}

	if selfURL == nil {
		log.Debugf("self url not found in request Link headers")
		http.Error(w, "Missing rel=self Link", http.StatusNotFound)
		return
	}

	// TODO: Verify we actually have a valid and confirmed Subscription for
	//       for the given topic (selfURL) with the Pod (hubEndpoint)
	//       to prevent abuse of the WebSub /notify endpoint

	ws.inbox <- &callback{topic: selfURL.String()}

	http.Error(w, "Notification Enqueued for Processing", http.StatusAccepted)
}

func (ws *WebSub) Stats() (stats WebSubStats) {
	ws.RLock()
	defer ws.RUnlock()

	stats.Topics = len(ws.subscribers)

	for _, subs := range ws.subscribers {
		stats.Subscribers += len(subs)
		for _, sub := range subs {
			if sub.Verified {
				stats.Verified++
			}
		}
	}

	stats.Subscriptions = len(ws.subscriptions)
	for _, sub := range ws.subscriptions {
		if sub.Confirmed() {
			stats.Confirmed++
		}
	}
	return
}

func (ws *WebSub) DebugEndpoint(w http.ResponseWriter, r *http.Request) {
	ws.RLock()
	defer ws.RUnlock()

	doc := struct {
		Endpoint      string
		Subscribers   map[string]Subscribers
		Subscriptions map[string]*Subscription
	}{
		Endpoint:      ws.endpoint,
		Subscribers:   ws.subscribers,
		Subscriptions: ws.subscriptions,
	}

	data, err := json.Marshal(doc)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (ws *WebSub) WebSubEndpoint(w http.ResponseWriter, r *http.Request) {
	log.Debug("WebSubEndpoint:")
	mode := r.FormValue("hub.mode")
	topic := r.FormValue("hub.topic")
	callback := r.FormValue("hub.callback")

	log.Debugf("mode: %s", mode)
	log.Debugf("topic: %s", topic)
	log.Debugf("callback: %s", callback)

	if strings.TrimSpace(callback) == "" {
		log.Errorf("no callback provided")
		http.Error(w, "No Callback", http.StatusBadRequest)
		return
	}

	if _, err := url.Parse(callback); err != nil {
		log.WithError(err).Errorf("error parsing callback %s", callback)
		http.Error(w, "Bad Callback", http.StatusBadRequest)
		return
	}

	if _, err := url.Parse(topic); err != nil {
		log.WithError(err).Errorf("error parsing topic %s", callback)
		http.Error(w, "Bad Topic", http.StatusBadRequest)
		return
	}
	if !ws.ValidateTopic(topic) {
		log.Debugf("invalid topic %q", topic)
		http.Error(w, "Invalid Topic", http.StatusBadRequest)
		return
	}

	switch strings.ToLower(mode) {

	case "subscribe":
		if !ws.HasSubscriberFor(topic, callback) {
			ws.AddSubscriber(NewSubscriber(topic, callback))
		}
		ws.verify <- &verification{
			target:       callback,
			topic:        topic,
			callback:     callback,
			challenge:    generateRandomChallengeString(),
			leaseSeconds: int(defaultWebSubLeaseTime.Seconds()),
		}
		http.Error(w, "Subscription Accepted", http.StatusAccepted)
		return

	case "unsubscribe":
		_, idx := ws.GetSubscriberFor(topic, callback)
		if idx != -1 {
			ws.DelSubscriber(topic, idx)
		}
		http.Error(w, "Subscription Removed", http.StatusAccepted)
		return
	}

	log.Debugf("invalid mode %q", mode)
	http.Error(w, "Invalid Mode", http.StatusBadRequest)
}

func (ws *WebSub) processInbox() {
	notification := <-ws.inbox

	if err := ws.Notify(notification.topic); err != nil {
		log.WithError(err).Errorf("error processing notification for %s", notification.topic)
	}
}

func (ws *WebSub) processOutbox() {
	notification := <-ws.outbox
	notification.attempts++

	if notification.attempts > defaultWebSubRedeliveryAttempts {
		log.Errorf(
			"giving up processing notification for topic=%s target=%s after %d attempts",
			notification.topic, notification.target, notification.attempts,
		)
		return
	}

	req, err := http.NewRequest(http.MethodPost, notification.target, nil)
	if err != nil {
		log.WithError(err).Errorf(
			"error creating notification request for topic=%s target=%s",
			notification.topic, notification.target,
		)
		return
	}

	req.Header.Add("Link", fmt.Sprintf(`<%s/websub>; rel="hub"`, ws.endpoint))
	req.Header.Add("Link", fmt.Sprintf(`<%s>; rel="self"`, notification.topic))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.WithError(err).Errorf(
			"error sending notification request for topic=%s target=%s",
			notification.topic, notification.target,
		)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusAccepted {
		log.Errorf(
			"bad response %s from callback for topic=%s target=%s",
			res.Status, notification.topic, notification.target,
		)
		return
	}

	log.Debugf("successfully sent notification to %s", notification.target)
}

func (ws *WebSub) processVerify() {
	verification := <-ws.verify
	verification.attempts++

	if verification.attempts > defaultWebSubRedeliveryAttempts {
		log.Errorf(
			"giving up processing verificationg for topic=%s callbac=%s after %d attempts",
			verification.topic, verification.callback, verification.attempts,
		)

		_, idx := ws.GetSubscriberFor(verification.topic, verification.callback)
		if idx != -1 {
			ws.DelSubscriber(verification.topic, idx)
		}

		return
	}

	req, err := http.NewRequest(http.MethodGet, verification.target, nil)
	if err != nil {
		log.WithError(err).Errorf(
			"error creating verification request topic=%s callbac=%s",
			verification.topic, verification.callback,
		)
		return
	}

	req.Header.Add("Link", fmt.Sprintf(`<%s/websub>; rel="hub"`, ws.endpoint))
	req.Header.Add("Link", fmt.Sprintf(`<%s>; rel="self"`, verification.topic))

	qs := url.Values{}
	qs.Set("hub.mode", "subscribe")
	qs.Set("hub.topic", verification.topic)
	qs.Set("hub.challenge", verification.challenge)
	qs.Set("hub.lease_seconds", fmt.Sprintf("%d", verification.leaseSeconds))
	req.URL.RawQuery = qs.Encode()
	log.Debugf("Sending websub verification request to %s", verification.target)
	log.Debugf("req.URL.Query(): %q", req.URL.Query())

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.WithError(err).Errorf(
			"error sending verification request topic=%s callbac=%s",
			verification.topic, verification.callback,
		)
		return
	}
	defer res.Body.Close()

	if res.StatusCode/100 != 2 {
		log.Errorf(
			"bad response %s from verification for topic=%s callbac=%s",
			res.Status, verification.topic, verification.callback,
		)
		return
	}

	log.Debugf("successfully sent verification to %s", verification.target)

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.WithError(err).Errorf(
			"error reading verification response for topic=%s callback=%s",
			verification.topic, verification.callback,
		)
		return
	}
	response := strings.TrimSpace(string(body))
	if response != verification.challenge {
		log.Debugf(
			"challenge verification failed for topic=%s callbac=%s %q != %q",
			verification.topic, verification.callback,
			response, verification.challenge,
		)
		return
	}

	subscriber, idx := ws.GetSubscriberFor(verification.topic, verification.callback)
	if idx == -1 {
		log.Errorf("no subscriber found for topic=%s callback=%s but verification was sent?!", verification.topic, verification.callback)
		return
	}
	subscriber.Verified = true
	subscriber.ExpiresAt = time.Now().Add(time.Duration(verification.leaseSeconds) * time.Second)
	log.Debugf("successfully verified subscriber for topic=%s callback=%s", verification.topic, verification.callback)
}
