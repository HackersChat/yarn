package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"mime"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	sync "github.com/sasha-s/go-deadlock"
	log "github.com/sirupsen/logrus"
	"go.yarn.social/types"
)

// FIXME: Fix this

const (
	defaultPeeringRefreshInterval = time.Hour * 24
	defaultPeeringDeadTimeout     = defaultPeeringRefreshInterval * 3
)

// Peer represents a peer
type Peer struct {
	URI              string `json:"uri"`
	Name             string `json:"name"`
	Logo             string `json:"logo"`
	Description      string `json:"description"`
	SoftwareVersion  string `json:"software_version"`
	BuildInformation string `json:"build_info"`

	// lastSeen records the timestamp of when we last saw this pod.
	LastSeen time.Time `json:"last_seen"`

	// lastUpdated is used to periodically re-check the peering pod's /info endpoint in case of changes.
	LastUpdated time.Time `json:"last_updated"`

	// Trusted indicates if the pod operator accepted this pod as a trusted
	// peer or not. Only when accepted, Pod Gossiping will take this peer into
	// consideration.
	Trusted bool `json:"trusted"`
}

// String returns a human-readable string representation of the Pod.
func (p *Peer) String() string {
	return fmt.Sprintf("Peer{Name: %s URI: %s}", p.Name, p.URI)
}

// PodInfo is an alias for Pod
// XXX: Type aliases for backwards compatibility with Cache v19
type PodInfo Peer

// IsZero returns true if the Pod is nil or both its Name and SoftwareVersion are empty strings,
// indicating that the Pod is uninitialized or has no meaningful data.
func (p *Peer) IsZero() bool {
	return (p == nil) || (p.Name == "" && p.SoftwareVersion == "")
}

// ShouldRefresh returns true if the Pod should be refreshed (i.e. its /info should be requested and updated)
// based on the last time it was updated.
func (p *Peer) ShouldRefresh() bool {
	return time.Since(p.LastUpdated) > defaultPeeringRefreshInterval
}

// makeJSONRequest sends an HTTP GET request to the specified path on the Pod's URI,
// expecting a JSON response. It sets the "Accept" header to "application/json" and
// checks that the response has a 2xx status code and a "Content-Type" of "application/json".
// Returns the response body as a byte slice or an error if the request fails or the response
// is not JSON.
func (p *Peer) makeJSONRequest(conf *Config, path string) ([]byte, error) {
	headers := make(http.Header)
	headers.Set("Accept", "application/json")

	res, err := RequestHTTP(conf, http.MethodGet, p.URI+path, headers)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode/100 != 2 {
		return nil, fmt.Errorf("non-success HTTP %s response for %s%s", res.Status, p.URI, path)
	}

	if ctype := res.Header.Get("Content-Type"); ctype != "" {
		mediaType, _, err := mime.ParseMediaType(ctype)
		if err != nil {
			return nil, err
		}
		if mediaType != "application/json" {
			return nil, fmt.Errorf("non-JSON response content type '%s' for %s%s", ctype, p.URI, path)
		}
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// GetTwt requests a twt from the peer pod by its hash and returns the decoded twt
// or an error. If the peer pod is not trusted, it returns an error.
func (p *Peer) GetTwt(conf *Config, hash string) (types.Twt, error) {
	if !p.Trusted {
		return nil, errors.New("untrusted peer")
	}

	data, err := p.makeJSONRequest(conf, "/twt/"+hash)
	if err != nil {
		return nil, err
	}

	twt, err := types.DecodeJSON(data)
	if err != nil {
		return nil, err
	}

	return twt, nil
}

// Peers is a sortable list of Peers
type Peers []*Peer

func (peers Peers) Len() int           { return len(peers) }
func (peers Peers) Less(i, j int) bool { return strings.Compare(peers[i].Name, peers[j].Name) < 0 }
func (peers Peers) Swap(i, j int)      { peers[i], peers[j] = peers[j], peers[i] }

// Peering manages the lifecycle of peering.
type Peering struct {
	mu       sync.RWMutex
	peers    map[string]*Peer // keyed by the pod's base URL
	conf     *Config
	stopChan chan struct{}
	ticker   *time.Ticker
}

// NewPeerManager constructs a new PeerManager.
func NewPeerManager(conf *Config) *Peering {
	return &Peering{
		peers:    make(map[string]*Peer),
		conf:     conf,
		stopChan: make(chan struct{}),
	}
}

// LoadFromFile loads a list of peers from a file.
func (pm *Peering) LoadFromFile(filename string) error {
	// Read the file and decode each peer into a Peer object.
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	var peers Peers
	if err := json.NewDecoder(file).Decode(&peers); err != nil {
		return err
	}

	pm.mu.Lock()
	for _, p := range peers {
		pm.peers[p.URI] = p
	}
	pm.mu.Unlock()

	return nil
}

// SaveToFile saves the list of peers to a file.
func (pm *Peering) SaveToFile(filename string) error {
	// Serialize the list of peers into JSON and write it to the file.
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	var peers Peers

	pm.mu.RLock()
	for _, p := range pm.peers {
		peers = append(peers, p)
	}
	pm.mu.RUnlock()

	return json.NewEncoder(file).Encode(peers)
}

// AddOrUpdatePeer adds a new peer or updates an existing one.
// If the peer already exists, its Trusted status is preserved.
func (pm *Peering) AddOrUpdatePeer(p *Peer) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if existing, ok := pm.peers[p.URI]; ok {
		// Preserve Trusted status if already set.
		p.Trusted = existing.Trusted
	}
	pm.peers[p.URI] = p
	log.Infof("Peer added/updated: %s", p)
}

// GetCandidatePeers returns a subset of trusted peers in random order.
// This ensures that we don't query all peers in a predictable order.
// And also ensures we don't overload the peering network with too many requests.
func (pm *Peering) GetCandidatePeers() Peers {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var peers Peers
	for _, p := range pm.peers {
		if p.Trusted {
			peers = append(peers, p)
		}
	}

	// Cut the peers in 1/2
	mid := len(peers) / 2
	peers = peers[:mid]

	// Shuffle the peers in random order.
	rand.Shuffle(len(peers), func(i, j int) {
		peers[i], peers[j] = peers[j], peers[i]
	})

	return peers
}

// ListPeers returns a slice of the current peers.
func (pm *Peering) ListPeers() Peers {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	var list Peers
	for _, p := range pm.peers {
		list = append(list, p)
	}
	// Optionally, sort the peers for consistent display order.
	sort.Sort(list)
	return list
}

// GetPeer retrieves a peer by URI.
func (pm *Peering) GetPeer(uri string) (*Peer, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	p, ok := pm.peers[uri]
	return p, ok
}

// DeletePeer removes a peer from management.
func (pm *Peering) DeletePeer(uri string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.peers, uri)
	log.Infof("Peer deleted: %s", uri)
}

// TrustPeer sets the Trusted flag for a given peer.
func (pm *Peering) TrustPeer(uri string, trusted bool) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	p, ok := pm.peers[uri]
	if !ok {
		return fmt.Errorf("peer %s not found", uri)
	}
	p.Trusted = trusted
	log.Infof("Peer %s trust set to %t", uri, trusted)
	return nil
}

// RefreshPeer re-fetches /info for a given peer if needed and updates it.
func (pm *Peering) RefreshPeer(uri string) error {
	pm.mu.RLock()
	p, ok := pm.peers[uri]
	pm.mu.RUnlock()
	if !ok {
		return fmt.Errorf("peer %s not found", uri)
	}
	// Only refresh if the peer's info is stale.
	if !p.ShouldRefresh() {
		return nil
	}

	// Prepare request to the peer's /info endpoint.
	headers := make(http.Header)
	headers.Set("Accept", "application/json")
	res, err := RequestHTTP(pm.conf, http.MethodGet, p.URI+"/info", headers)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode/100 != 2 {
		return fmt.Errorf("non-success HTTP response: %s", res.Status)
	}
	if ctype := res.Header.Get("Content-Type"); ctype != "" {
		mediaType, _, err := mime.ParseMediaType(ctype)
		if err != nil {
			return err
		}
		if mediaType != "application/json" {
			return fmt.Errorf("unexpected content type: %s", ctype)
		}
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	var newPeer Peer
	if err := json.Unmarshal(data, &newPeer); err != nil {
		return err
	}
	newPeer.URI = p.URI
	newPeer.LastUpdated = time.Now()
	newPeer.LastSeen = time.Now()
	// Preserve the Trusted flag from the existing peer.
	newPeer.Trusted = p.Trusted

	pm.mu.Lock()
	pm.peers[uri] = &newPeer
	pm.mu.Unlock()

	log.Infof("Peer refreshed: %s", newPeer.String())
	return nil
}

// RefreshAllPeers iterates over all peers and refreshes those whose info is stale.
func (pm *Peering) RefreshAllPeers() {
	pm.mu.RLock()
	var uris []string
	for uri, p := range pm.peers {
		if p.ShouldRefresh() {
			uris = append(uris, uri)
		}
	}
	pm.mu.RUnlock()

	for _, uri := range uris {
		if err := pm.RefreshPeer(uri); err != nil {
			log.WithError(err).Warnf("error refreshing peer %s", uri)
		}
	}
}

// CleanupDeadPeers removes peers that have not been seen for a specified duration.
func (pm *Peering) CleanupDeadPeers() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	for uri, p := range pm.peers {
		if time.Since(p.LastUpdated) > defaultPeeringDeadTimeout {
			trustedStatus := "untrusted"
			if p.Trusted {
				trustedStatus = "trusted"
			}
			log.Infof("removing dead %s peer '%s' (%s), last seen: %s (%s ago), last updated: %s (%s ago)",
				trustedStatus, p.Name, p.URI,
				p.LastSeen.Format(time.RFC3339),
				time.Since(p.LastSeen).Round(time.Second),
				p.LastUpdated.Format(time.RFC3339),
				time.Since(p.LastUpdated).Round(time.Second))
			delete(pm.peers, uri)
		}
	}
}

// Start launches a background goroutine that periodically refreshes peer information.
// It uses the refresh interval from the config. If not set, a default of one hour is used.
func (pm *Peering) Start() {
	refreshInterval := defaultPeeringRefreshInterval
	if refreshInterval <= 0 {
		refreshInterval = time.Hour // default refresh interval
	}
	pm.ticker = time.NewTicker(refreshInterval)
	go func() {
		for {
			select {
			case <-pm.ticker.C:
				pm.RefreshAllPeers()
			case <-pm.stopChan:
				pm.ticker.Stop()
				return
			}
		}
	}()
	log.Infof("Peering started with refresh interval %s", refreshInterval)
}

// Stop terminates the background peer refresher.
func (pm *Peering) Stop() {
	close(pm.stopChan)
	log.Info("PeerManager stopped")
}

// PeerDetector is responsible for detecting remote pods from HTTP requests and responses.
type PeerDetector struct {
	conf    *Config
	peering *Peering
}

// NewPeerDetector creates a new PeerDetector with the given configuration and peering manager.
func NewPeerDetector(conf *Config, peering *Peering) *PeerDetector {
	return &PeerDetector{
		conf:    conf,
		peering: peering,
	}
}

// DetectFromResponse examines an HTTP response for evidence that the sender is a pod.
// For example, it can check the "Powered-By" header. If a valid pod is detected,
// it creates a minimal Pod object and updates the peering manager.
func (pd *PeerDetector) DetectFromResponse(res *http.Response) {
	if res == nil {
		return
	}

	poweredBy := res.Header.Get("Powered-By")
	if poweredBy == "" {
		return
	}

	// Parse the header into a structured user agent.
	ua, err := ParseUserAgent(poweredBy)
	if err != nil {
		return
	}

	// Only proceed if the user agent indicates a pod.
	if !ua.IsPod() {
		return
	}

	podBaseURL := ua.PodBaseURL()
	if podBaseURL == "" {
		return
	}

	// If we already know this peer, ignore it.
	if _, ok := pd.peering.GetPeer(podBaseURL); ok {
		return
	}

	// Create minimal Peer object and update the peering manager.
	newPeer := Peer{URI: podBaseURL, LastSeen: time.Now()}
	pd.peering.AddOrUpdatePeer(&newPeer)

	log.Infof("found new peer %s in request: %s", podBaseURL, poweredBy)

	if err := pd.peering.RefreshPeer(podBaseURL); err != nil {
		pd.peering.DeletePeer(podBaseURL)
	}
}

// DetectFromRequest examines an incoming HTTP request's User-Agent header
// to determine whether the request came from a pod. If so, it updates the peering manager.
func (pd *PeerDetector) DetectFromRequest(req *http.Request) {
	if req == nil {
		return
	}

	uaStr := req.UserAgent()
	if uaStr == "" {
		return
	}

	ua, err := ParseUserAgent(uaStr)
	if err != nil {
		return
	}

	if !ua.IsPod() {
		return
	}

	podBaseURL := ua.PodBaseURL()
	if podBaseURL == "" {
		return
	}

	// If we already know this peer, ignore it.
	if _, ok := pd.peering.GetPeer(podBaseURL); ok {
		return
	}

	// Create minimal Peer object and update the peering manager.
	newPeer := Peer{URI: podBaseURL, LastSeen: time.Now()}
	pd.peering.AddOrUpdatePeer(&newPeer)

	log.Infof("found new peer %s in request: %s", podBaseURL, uaStr)

	if err := pd.peering.RefreshPeer(podBaseURL); err != nil {
		pd.peering.DeletePeer(podBaseURL)
	}
}
