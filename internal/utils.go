// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"image"
	"io"
	"math"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	text_template "text/template"
	"time"

	// Blank import so we can handle image/jpeg

	"image/gif"
	_ "image/jpeg"
	"image/png"

	"git.mills.io/prologic/go-gopher"
	"git.mills.io/yarnsocial/yarn"
	"github.com/PuerkitoBio/goquery"
	"github.com/audiolion/ipip"
	"github.com/disintegration/gift"
	"github.com/disintegration/imageorient"
	"github.com/dustin/go-humanize"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/goware/urlx"
	"github.com/h2non/filetype"
	"github.com/makeworld-the-better-one/go-gemini"
	"github.com/microcosm-cc/bluemonday"
	"github.com/rrivera/identicon"
	sync "github.com/sasha-s/go-deadlock"
	log "github.com/sirupsen/logrus"
	"go.mills.io/webfinger"
	"go.yarn.social/lextwt"
	"go.yarn.social/types"
	"golang.org/x/crypto/blake2b"
)

const (
	avatarsDir  = "avatars"
	externalDir = "external"
	mediaDir    = "media"

	me = "me"

	maxUsernameLength   = 15 // avg 6 chars / 2 syllables per name commonly
	maxFeedNameLength   = 25 // avg 4.7 chars per word in English so ~5 words
	maxTwtContextLength = 140

	DayAgo   = time.Hour * 24
	WeekAgo  = DayAgo * 7
	MonthAgo = DayAgo * 30
	YearAgo  = MonthAgo * 12
)

// TwtTextFormat represents the format of which the twt text gets formatted to
type TwtTextFormat int

const (
	// MarkdownFmt to use markdown format
	MarkdownFmt TwtTextFormat = iota
	// HTMLFmt to use HTML format
	HTMLFmt
	// TextFmt to use for og:description
	TextFmt
)

var (
	reservedUsernames = []string{
		me,
	}

	validFeedPath     = regexp.MustCompile(`^\/user\/(\S+)\/twtxt\.txt$`)
	validFeedName     = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)
	validUsername     = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]+$`)
	singleUserUARegex = regexp.MustCompile(`(.+) \(\+(https?://\S+/\S+); @(\S+)\)`)
	multiUserUARegex  = regexp.MustCompile(`(.+) \(~(https?://\S+\/\S+); contact=(https?://\S+)\)`)
	yarndUserUARegex  = regexp.MustCompile(`(.+) \(Pod: (\S+) Support: (https?://\S+)\)`)
	mediaURIRegex     = regexp.MustCompile(`\/media\/[a-zA-Z0-9]{16,}\.(png|gif|mp4|mp3)`)
	// yarnURIRegex      = regexp.MustCompile(`@?(\S+)@(\S+)`)

	ErrInvalidFeedName  = errors.New("error: invalid feed name")
	ErrBadRequest       = errors.New("error: request failed with non-200 response")
	ErrFeedNameTooLong  = errors.New("error: feed name is too long")
	ErrInvalidUsername  = errors.New("error: invalid username")
	ErrUsernameTooLong  = errors.New("error: username is too long")
	ErrInvalidUserAgent = errors.New("error: invalid twtxt user agent")
	ErrReservedUsername = errors.New("error: username is reserved")
	ErrInvalidImage     = errors.New("error: invalid image")
	ErrInvalidAudio     = errors.New("error: invalid audio")
	ErrInvalidVideo     = errors.New("error: invalid video")
	ErrInvalidVideoSize = errors.New("error: invalid video size")
)

func GenerateRandomToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func DecodeHash(hash string) ([]byte, error) {
	encoding := base32.StdEncoding.WithPadding(base32.NoPadding)
	return encoding.DecodeString(strings.ToUpper(hash))
}

func FastHash(data []byte) string {
	sum := blake2b.Sum256(data)

	// Base32 is URL-safe, unlike Base64, and shorter than hex.
	encoding := base32.StdEncoding.WithPadding(base32.NoPadding)
	hash := strings.ToLower(encoding.EncodeToString(sum[:]))

	return hash
}

func FastHashString(s string) string {
	return FastHash([]byte(s))
}

func FastHashFile(fn string) (string, error) {
	data, err := os.ReadFile(fn)
	if err != nil {
		return "", err
	}
	return FastHash(data), nil
}

func IntPow(x, y int) int {
	return int(math.Pow(float64(x), float64(y)))
}

func Slugify(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		log.WithError(err).Warnf("Slugify(): error parsing uri: %s", uri)
		return ""
	}

	var buf bytes.Buffer
	lastWasDash := true
	for _, c := range strings.ToLower(fmt.Sprintf("%s-%s", u.Hostname(), u.Path)) {
		switch {
		case 'a' <= c && c <= 'z' || '0' <= c && c <= '9' || c == '-' || c == '_':
			buf.WriteRune(c)
			lastWasDash = c == '-'
		case c == '&':
			buf.WriteString("and")
			lastWasDash = false
		case c == '@':
			buf.WriteString("at")
			lastWasDash = false
		default:
			// don't generate consecutive dashes
			if !lastWasDash {
				buf.WriteRune('-')
			}
			lastWasDash = true
		}
	}

	return strings.Trim(buf.String(), "-")
}

func GenerateAvatar(conf *Config, domainNick string) (image.Image, error) {
	ig, err := identicon.New(conf.Name, 7, 4)
	if err != nil {
		log.WithError(err).Error("error creating identicon generator")
		return nil, err
	}

	ii, err := ig.Draw(domainNick)
	if err != nil {
		log.WithError(err).Errorf("error generating external avatar for %s", domainNick)
		return nil, err
	}

	return ii.Image(conf.AvatarResolution), nil
}

func ReplaceExt(fn, newExt string) string {
	oldExt := filepath.Ext(fn)
	return fmt.Sprintf("%s%s", strings.TrimSuffix(fn, oldExt), newExt)
}

func HasExternalAvatarChanged(conf *Config, twter types.Twter) bool {
	uri := NormalizeURL(twter.URI)
	slug := Slugify(uri)
	fn := filepath.Join(conf.Data, externalDir, fmt.Sprintf("%s.cbf", slug))

	log := log.WithField("uri", uri)

	// If the %s.cbf doesn't yet exist, then assume the external avatar has changed
	if !FileExists(fn) {
		return true
	}

	// If the twter.Avatar uri is empty but a %s.cbf exists then assume the avatar has not changed
	if twter.Avatar == "" {
		return false
	}

	// If the twter.Avatar uri cannot be parsed but a %s.cbf exists then assume the avatar has not changed
	u, err := url.Parse(twter.Avatar)
	if err != nil {
		log.WithError(err).Warnf("error parsing avatar url for %s", twter.Avatar)
		return false
	}

	// if we cannot read the %s.cbf file assume the avatar has not changed
	data, err := os.ReadFile(fn)
	if err != nil {
		log.WithError(err).Warnf("error reading avatar cbf for %s", slug)
		return false
	}

	// compare the cbf(s)
	return string(data) != FastHashString(u.String())
}

func GetExternalAvatar(conf *Config, twter types.Twter) {
	uri := NormalizeURL(twter.URI)
	slug := Slugify(uri)
	fn := filepath.Join(conf.Data, externalDir, fmt.Sprintf("%s.png", slug))

	log := log.WithField("uri", uri)

	//
	// Use an already cached Avatar (unless there's a new one!)
	//

	if FileExists(fn) && !HasExternalAvatarChanged(conf, twter) {
		return
	}

	// Use the Avatar advertised in the feed
	if twter.Avatar != "" {
		u, err := url.Parse(twter.Avatar)
		if err != nil {
			log.WithError(err).Errorf("error parsing avatar url %s", twter.Avatar)
			return
		}

		// If the URL is relative, construct a full URL based on the feed URL
		if !u.IsAbs() {
			fURI, err := url.Parse(uri)
			if err != nil {
				log.WithError(err).Errorf("error parsing feed url %s", uri)
				return
			}
			u.Scheme = fURI.Scheme
			u.Host = fURI.Host
			u.JoinPath(u.Path, twter.Avatar)
		}

		opts := &ImageOptions{Resize: true, Width: conf.AvatarResolution, Height: conf.AvatarResolution}
		if _, err := DownloadImage(conf, u.String(), externalDir, slug, opts); err != nil {
			log.WithError(err).Errorf("error downloading external avatar: %s", u)
			return
		}
		if err := os.WriteFile(ReplaceExt(fn, ".cbf"), []byte(FastHashString(u.String())), 0644); err != nil {
			log.WithError(err).Warnf("error writing avatar cbf for %s", slug)
		}
		return
	}
}

func RequestGemini(conf *Config, uri string) (*gemini.Response, error) {
	res, err := gemini.Fetch(uri)
	if err != nil {
		log.WithError(err).Errorf("%s: gemini.Fetch fail: %s", uri, err)
		return nil, err
	}

	if res.Status != gemini.StatusSuccess {
		return nil, fmt.Errorf("non-success gemini %d response for %s", res.Status, uri)
	}

	return res, nil
}

func RequestGopher(conf *Config, uri string) (*gopher.Response, error) {
	res, err := gopher.Get(uri)
	if err != nil {
		log.WithError(err).Errorf("%s: gopher.Get fail: %s", uri, err)
		return nil, err
	}

	if res.Type != gopher.FILE {
		return nil, fmt.Errorf("unexpected type %s (expected FILE)", res.Type)
	}

	return res, nil
}

func RequestHTTP(conf *Config, method, url string, headers http.Header) (*http.Response, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		log.WithError(err).Errorf("%s: http.NewRequest fail: %s", url, err)
		return nil, err
	}

	if headers == nil {
		headers = make(http.Header)
	}

	// Set a default User-Agent (if none set)
	if headers.Get("User-Agent") == "" {
		headers.Set(
			"User-Agent",
			fmt.Sprintf(
				"yarnd/%s (Pod: %s Support: %s)",
				yarn.FullVersion(), conf.Name, URLForPage(conf.BaseURL, "support"),
			),
		)
	}

	req.Header = headers

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.WithError(err).Errorf("%s: client.Do fail: %s", url, err)
		return nil, err
	}

	return res, nil
}

func ResourceExists(conf *Config, url string) bool {
	res, err := RequestHTTP(conf, http.MethodHead, url, nil)
	if err != nil {
		log.WithError(err).Errorf("error checking if %s exists", url)
		return false
	}
	defer res.Body.Close()

	return res.StatusCode/100 == 2
}

func LineCount(r io.Reader) (int, error) {
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count, nil

		case err != nil:
			return count, err
		}
	}
}

func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// CmdExists ...
func CmdExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// RunCmd ...
func RunCmd(timeout time.Duration, command string, args ...string) error {
	var (
		ctx    context.Context
		cancel context.CancelFunc
	)

	if timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if ws, ok := exitError.Sys().(syscall.WaitStatus); ok && ws.Signal() == syscall.SIGKILL {
				err = &ErrCommandKilled{Err: err, Signal: ws.Signal()}
			} else {
				err = &ErrCommandFailed{Err: err, Status: exitError.ExitCode()}
			}
		}

		log.
			WithError(err).
			WithField("out", string(out)).
			Errorf("error running command")

		return err
	}

	return nil
}

// RenderLogo ...
func RenderLogo(logo string, podName string) (template.HTML, error) {
	t := text_template.Must(text_template.New("logo").Parse(logo))
	buf := bytes.NewBuffer([]byte{})
	err := t.Execute(buf, map[string]string{"PodName": podName})
	if err != nil {
		return "", err
	}

	return template.HTML(buf.String()), nil
}

// RenderCSS ...
func RenderCSS(css string) (template.CSS, error) {
	t := text_template.Must(text_template.New("css").Parse(css))
	buf := bytes.NewBuffer([]byte{})
	err := t.Execute(buf, map[string]string{"PodCSS": css})
	if err != nil {
		return "", err
	}

	return template.CSS(buf.String()), nil
}

// RenderJS ...
func RenderJS(js string) (template.JS, error) {
	t := text_template.Must(text_template.New("js").Parse(js))
	buf := bytes.NewBuffer([]byte{})
	err := t.Execute(buf, map[string]string{"PodJS": js})
	if err != nil {
		return "", err
	}

	return template.JS(buf.String()), nil
}

func IsLocalURLFactory(conf *Config) func(url string) bool {
	return func(url string) bool {
		normalizedURL := NormalizeURL(url)
		if normalizedURL == "" {
			return false
		}
		return strings.HasPrefix(normalizedURL, NormalizeURL(conf.BaseURL))
	}
}

func GetUserFromTwter(conf *Config, db Store, twter types.Twter) (*User, error) {
	if !strings.HasPrefix(twter.URI, conf.BaseURL) {
		return nil, fmt.Errorf("error: %s does not match our base url of %s", twter.URI, conf.BaseURL)
	}

	userURL := UserURL(twter.URI)
	username := filepath.Base(userURL)

	return db.GetUser(username)
}

func GetUserFromURL(conf *Config, db Store, url string) (*User, error) {
	if !strings.HasPrefix(url, conf.BaseURL) {
		return nil, fmt.Errorf("error: %s does not match our base url of %s", url, conf.BaseURL)
	}

	userURL := UserURL(url)
	username := filepath.Base(userURL)

	return db.GetUser(username)
}

func WebMention(target, source string) error {
	targetURL, err := url.Parse(target)
	if err != nil {
		log.WithError(err).Error("error parsing target url")
		return err
	}
	sourceURL, err := url.Parse(source)
	if err != nil {
		log.WithError(err).Error("error parsing source url")
		return err
	}
	webmentions.Notify(targetURL, sourceURL)
	return nil
}

func MapKeys[K comparable, V any](kv map[K]V) []K {
	var res []K
	for k := range kv {
		res = append(res, k)
	}
	return res
}

func MapValues[K comparable, V any](kv map[K]V) []V {
	var res []V
	for _, v := range kv {
		res = append(res, v)
	}
	return res
}

func MapStrings(xs []string, f func(s string) string) []string {
	var res []string
	for _, x := range xs {
		res = append(res, f(x))
	}
	return res
}

func HasString(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

func UniqStrings(xs []string) []string {
	set := make(map[string]bool)
	for _, x := range xs {
		if _, ok := set[x]; !ok {
			set[x] = true
		}
	}

	res := []string{}
	for k := range set {
		res = append(res, k)
	}
	return res
}

func RemoveString(xs []string, e string) []string {
	res := []string{}
	for _, x := range xs {
		if x == e {
			continue
		}
		res = append(res, x)
	}
	return res
}

func UniqueKeyFor(kv map[string]string, k string) string {
	K := k
	for i := 1; i < 99; i++ {
		if _, ok := kv[K]; !ok {
			return K
		}
		K = fmt.Sprintf("%s_%d", k, i)
	}
	return fmt.Sprintf("%s_???", k)
}

func IsGifImage(fn string) bool {
	f, err := os.Open(fn)
	if err != nil {
		log.WithError(err).Warnf("error opening file %s", fn)
		return false
	}
	defer f.Close()

	head := make([]byte, 261)
	if _, err := f.Read(head); err != nil {
		log.WithError(err).Warnf("error reading from file %s", fn)
		return false
	}

	imageType, err := filetype.Image(head)
	if err != nil {
		return false
	}
	return imageType.MIME.Type == "image" && imageType.MIME.Subtype == "gif"
}

func IsImage(fn string) bool {
	f, err := os.Open(fn)
	if err != nil {
		log.WithError(err).Warnf("error opening file %s", fn)
		return false
	}
	defer f.Close()

	head := make([]byte, 261)
	if _, err := f.Read(head); err != nil {
		log.WithError(err).Warnf("error reading from file %s", fn)
		return false
	}

	return filetype.IsImage(head)
}

func IsAudio(fn string) bool {
	f, err := os.Open(fn)
	if err != nil {
		log.WithError(err).Warnf("error opening file %s", fn)
		return false
	}
	defer f.Close()

	head := make([]byte, 261)
	if _, err := f.Read(head); err != nil {
		log.WithError(err).Warnf("error reading from file %s", fn)
		return false
	}

	return filetype.IsAudio(head)
}

func IsVideo(fn string) bool {
	f, err := os.Open(fn)
	if err != nil {
		log.WithError(err).Warnf("error opening file %s", fn)
		return false
	}
	defer f.Close()

	head := make([]byte, 261)
	if _, err := f.Read(head); err != nil {
		log.WithError(err).Warnf("error reading from file %s", fn)
		return false
	}

	return filetype.IsVideo(head)
}

type ImageOptions struct {
	Resize bool
	Width  int
	Height int
}

type AudioOptions struct {
	Resample   bool
	Channels   int
	Samplerate int
	Bitrate    int
}

type VideoOptions struct {
	Resize bool
	Size   int
}

func DownloadImage(conf *Config, url string, resource, name string, opts *ImageOptions) (string, error) {
	res, err := http.Get(url)
	if err != nil {
		log.WithError(err).Errorf("error downloading image from %s", url)
		return "", err
	}

	tf, err := receiveFile(res.Body, conf.MaxUploadSize, "yarnd-image-*")
	if err != nil {
		return "", err
	}
	_, _ = io.Copy(io.Discard, res.Body)
	_ = res.Body.Close()

	defer os.Remove(tf.Name())

	if !IsImage(tf.Name()) {
		return "", ErrInvalidImage
	}

	if _, err := tf.Seek(0, io.SeekStart); err != nil {
		log.WithError(err).Error("error seeking temporary file")
		return "", err
	}

	return ProcessImage(conf, tf.Name(), resource, name, opts)
}

func ReceiveAudio(r io.Reader, maxLimit int64) (string, error) {
	tf, err := receiveFile(r, maxLimit, "yarnd-audio-*")
	if err != nil {
		return "", err
	}

	if !IsAudio(tf.Name()) {
		return "", ErrInvalidAudio
	}

	return tf.Name(), nil
}

func ReceiveImage(r io.Reader, maxLimit int64) (string, error) {
	tf, err := receiveFile(r, maxLimit, "yarnd-image-*")
	if err != nil {
		return "", err
	}

	if !IsImage(tf.Name()) {
		return "", ErrInvalidImage
	}

	return tf.Name(), nil
}

func ReceiveVideo(r io.Reader, maxLimit int64) (string, error) {
	tf, err := receiveFile(r, maxLimit, "yarnd-video-*")
	if err != nil {
		return "", err
	}

	if !IsVideo(tf.Name()) {
		return "", ErrInvalidVideo
	}

	return tf.Name(), nil
}

func receiveFile(r io.Reader, maxLimit int64, filePattern string) (*os.File, error) {
	tf, err := os.CreateTemp("", filePattern)
	if err != nil {
		log.WithError(err).Error("error creating temporary file")
		return nil, err
	}

	if _, err := io.Copy(tf, io.LimitReader(r, maxLimit)); err != nil {
		log.WithError(err).Error("error writing temporary file")
		return tf, err
	}

	if _, err := tf.Seek(0, io.SeekStart); err != nil {
		log.WithError(err).Error("error seeking temporary file")
		return tf, err
	}

	return tf, nil
}

func copyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func TranscodeAudio(conf *Config, ifn string, resource, name string, opts *AudioOptions) (string, error) {
	defer os.Remove(ifn)

	p := filepath.Join(conf.Data, resource)
	if err := os.MkdirAll(p, 0755); err != nil {
		log.WithError(err).Errorf("error creating %s directory", resource)
		return "", err
	}

	var ofn string

	if name == "" {
		ofn = filepath.Join(p, fmt.Sprintf("%s.mp3", GenerateRandomToken()))
	} else {
		ofn = fmt.Sprintf("%s.mp3", filepath.Join(p, name))
	}

	of, err := os.OpenFile(ofn, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.WithError(err).Error("error opening output file")
		return "", err
	}
	defer of.Close()

	wg := sync.WaitGroup{}

	TranscodeMP3 := func(ctx context.Context, errs chan error) {
		defer wg.Done()

		if err := RunCmd(
			conf.TranscoderTimeout,
			"ffmpeg",
			"-y",
			"-i", ifn,
			"-acodec", "mp3",
			"-strict", "-2",
			"-loglevel", "quiet",
			ReplaceExt(ofn, ".mp3"),
		); err != nil {
			log.WithError(err).Error("error transcoding video")
			errs <- err
			return
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var finalErr error

	nErrors := 0
	errChan := make(chan error)

	wg.Add(1)

	go TranscodeMP3(ctx, errChan)

	go func(ctx context.Context) {
		for {
			select {
			case err, ok := <-errChan:
				if !ok {
					return
				}
				nErrors++
				log.WithError(err).Errorf("TranscodeVideo() error")

				if errors.Is(err, &ErrCommandKilled{}) {
					finalErr = &ErrTranscodeTimeout{Err: err}
				} else {
					finalErr = &ErrTranscodeFailed{Err: err}
				}
			case <-ctx.Done():
				return
			}
		}
	}(ctx)

	wg.Wait()
	close(errChan)

	if nErrors > 0 {
		err = &ErrAudioUploadFailed{Err: finalErr}
		log.WithError(err).Error("TranscodeAudio() too many errors")
		return "", err
	}

	return fmt.Sprintf(
		"%s/%s/%s",
		strings.TrimSuffix(conf.BaseURL, "/"),
		resource, filepath.Base(ofn),
	), nil
}

func ResizeGif(srcFile string, width int, height int) (*gif.GIF, error) {
	f, err := os.Open(srcFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	im, err := gif.DecodeAll(f)
	if err != nil {
		return nil, err
	}

	if width == 0 {
		width = int(im.Config.Width * height / im.Config.Width)
	} else if height == 0 {
		height = int(width * im.Config.Height / im.Config.Width)
	}

	// reset the gif width and height
	im.Config.Width = width
	im.Config.Height = height

	g := gift.New(
		gift.Resize(width, height, gift.LanczosResampling),
	)

	newImages := []*image.Paletted{}
	for _, i := range im.Image {
		dst := image.NewPaletted(g.Bounds(i.Bounds()), i.Palette)
		g.Draw(dst, i)
		newImages = append(newImages, dst)
	}
	im.Image = newImages

	return im, nil
}

// Save gif file
func SaveGif(gifImg *gif.GIF, desFile string) error {
	f, err := os.Create(desFile)
	if err != nil {
		return err
	}
	defer f.Close()

	return gif.EncodeAll(f, gifImg)
}

func ProcessImage(conf *Config, ifn string, resource, name string, opts *ImageOptions) (string, error) {
	defer os.Remove(ifn)

	p := filepath.Join(conf.Data, resource)
	if err := os.MkdirAll(p, 0755); err != nil {
		log.WithError(err).Error("error creating avatars directory")
		return "", err
	}

	var (
		ofn string
		tfn string
	)

	isGIF := IsGifImage(ifn)

	var ext string

	if isGIF {
		ext = "gif"
	} else {
		ext = "png"
	}

	if name == "" {
		token := GenerateRandomToken()
		tfn = filepath.Join(p, fmt.Sprintf("%s.%s", token, ext))
		ofn = filepath.Join(p, fmt.Sprintf("%s.orig.%s", token, ext))
	} else {
		tfn = fmt.Sprintf("%s.%s", filepath.Join(p, name), ext)
		ofn = fmt.Sprintf("%s.orig.%s", filepath.Join(p, name), ext)
	}

	if _, err := copyFile(ifn, ofn); err != nil {
		log.WithError(err).Error("error copying input file")
		return "", err
	}

	if isGIF {
		gif, err := ResizeGif(ifn, opts.Width, opts.Height)
		if err != nil {
			log.WithError(err).Error("error downscaling GIF image")
			return "", err
		}
		if err := SaveGif(gif, tfn); err != nil {
			log.WithError(err).Error("error encoding GIF image")
			return "", err
		}
	} else {
		f, err := os.Open(ifn)
		if err != nil {
			log.WithError(err).Error("error opening input file")
			return "", err
		}
		defer f.Close()

		img, _, err := imageorient.Decode(f)
		if err != nil {
			log.WithError(err).Error("imageorient.Decode failed")
			return "", err
		}

		g := gift.New()

		if opts != nil && opts.Resize {
			if opts.Width > 0 && opts.Height > 0 {
				g.Add(gift.ResizeToFit(opts.Width, opts.Height, gift.LanczosResampling))
			} else if (opts.Width+opts.Height > 0) && (opts.Height > 0 || img.Bounds().Size().X > opts.Width) {
				g.Add(gift.Resize(opts.Width, opts.Height, gift.LanczosResampling))
			}
		}

		newImg := image.NewRGBA(g.Bounds(img.Bounds()))

		g.Draw(newImg, img)

		of, err := os.OpenFile(tfn, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.WithError(err).Error("error opening thumbnail file")
			return "", err
		}
		defer of.Close()

		if err := png.Encode(of, newImg); err != nil {
			log.WithError(err).Error("error encoding image")
			return "", err
		}
	}

	return fmt.Sprintf(
		"%s/%s/%s",
		strings.TrimSuffix(conf.BaseURL, "/"),
		resource, filepath.Base(tfn),
	), nil
}

func TranscodeVideo(conf *Config, ifn string, resource, name string, opts *VideoOptions) (string, error) {
	defer os.Remove(ifn)

	p := filepath.Join(conf.Data, resource)
	if err := os.MkdirAll(p, 0755); err != nil {
		log.WithError(err).Errorf("error creating %s directory", resource)
		return "", err
	}

	var ofn string

	if name == "" {
		ofn = filepath.Join(p, fmt.Sprintf("%s.mp4", GenerateRandomToken()))
	} else {
		ofn = fmt.Sprintf("%s.mp4", filepath.Join(p, name))
	}

	of, err := os.OpenFile(ofn, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.WithError(err).Error("error opening output file")
		return "", err
	}
	defer of.Close()

	wg := sync.WaitGroup{}

	TranscodeMP4 := func(ctx context.Context, errs chan error) {
		defer wg.Done()

		if err := RunCmd(
			conf.TranscoderTimeout,
			"ffmpeg",
			"-y",
			"-i", ifn,
			"-r", "24",
			"-preset", "ultrafast",
			"-vcodec", "h264",
			"-acodec", "aac",
			"-strict", "-2",
			"-loglevel", "quiet",
			ofn,
		); err != nil {
			log.WithError(err).Error("error transcoding video")
			errs <- err
			return
		}
	}

	GeneratePoster := func(ctx context.Context, errs chan error) {
		defer wg.Done()

		// ffmpeg -ss 00:00:03.000 -i video.mp4 -y -vframes 1 -strict -loglevel quiet poster.png
		// ffmpeg i video.mp4 -y -vf thumbnail -t 3 -vframes 1 -strict -loglevel quiet poster.png
		if err := RunCmd(
			conf.TranscoderTimeout,
			"ffmpeg",
			"-i", ifn,
			"-y",
			"-vf", "thumbnail",
			"-t", "3",
			"-vframes", "1",
			"-strict", "-2",
			"-loglevel", "quiet",
			ReplaceExt(ofn, ".png"),
		); err != nil {
			log.WithError(err).Error("error generating video poster")
			errs <- err
			return
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var finalErr error

	nErrors := 0
	errChan := make(chan error)

	wg.Add(2)

	go TranscodeMP4(ctx, errChan)
	go GeneratePoster(ctx, errChan)

	go func(ctx context.Context) {
		for {
			select {
			case err, ok := <-errChan:
				if !ok {
					return
				}
				nErrors++
				log.WithError(err).Errorf("TranscodeVideo() error")

				if errors.Is(err, &ErrCommandKilled{}) {
					finalErr = &ErrTranscodeTimeout{Err: err}
				} else {
					finalErr = &ErrTranscodeFailed{Err: err}
				}
			case <-ctx.Done():
				return
			}
		}
	}(ctx)

	wg.Wait()
	close(errChan)

	if nErrors > 0 {
		err = &ErrVideoUploadFailed{Err: finalErr}
		log.WithError(err).Error("TranscodeVideo() too many errors")
		return "", err
	}

	return fmt.Sprintf(
		"%s/%s/%s",
		strings.TrimSuffix(conf.BaseURL, "/"),
		resource, filepath.Base(ofn),
	), nil
}

func StoreUploadedImage(conf *Config, r io.Reader, resource, name string, opts *ImageOptions) (string, error) {
	fn, err := ReceiveImage(r, conf.MaxUploadSize)
	if err != nil {
		log.WithError(err).Error("error receiving image")
		return "", err
	}

	return ProcessImage(conf, fn, resource, name, opts)
}

func NormalizeFeedName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ToLower(name)
	return name
}

func ValidateFeed(conf *Config, profile types.Profile, uri string) (*types.Twter, error) {
	var body io.ReadCloser

	if strings.HasPrefix(uri, "gopher://") {
		res, err := RequestGopher(conf, uri)
		if err != nil {
			return nil, err
		}
		body = res.Body
	} else if strings.HasPrefix(uri, "gemini://") {
		res, err := RequestGemini(conf, uri)
		if err != nil {
			return nil, err
		}
		body = res.Body
	} else if strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
		res, err := RequestHTTP(conf, http.MethodGet, uri, nil)
		if err != nil {
			return nil, err
		}
		if res.StatusCode != 200 {
			return nil, ErrBadRequest
		}
		body = res.Body
	} else {
		client := webfinger.NewClient(nil)
		client.AllowHTTP = true

		log.Debugf("performing webfinger lookup for: %s", uri)
		jrd, err := client.Lookup(strings.TrimPrefix(uri, "@"), nil)
		if err != nil {
			return nil, fmt.Errorf("error looking up %s via webfinger: %w", uri, err)
		}

		found := false
		for _, link := range jrd.Links {
			log.Debugf("link rel=%s type=%s", link.Rel, link.Type)
			if link.Rel == webfinger.RelSelf {
				if link.Type == "text/plain" {
					log.Debugf("found link rel=%s type=%s", link.Rel, link.Type)
					return ValidateFeed(conf, profile, link.Href)
				}
			}
		}
		if !found {
			return nil, fmt.Errorf("could not find a valid rel=self link for %s", uri)
		}
	}

	defer body.Close()

	limitedReader := &io.LimitedReader{R: body, N: conf.MaxFetchLimit}
	twter := types.Twter{URI: uri}
	tf, err := types.ParseFile(limitedReader, &twter)
	if err != nil {
		return nil, err
	}

	return tf.Twter(), nil
}

func ValidateFeedName(path string, name string) error {
	if !validFeedName.MatchString(name) {
		return ErrInvalidFeedName
	}
	if len(name) > maxFeedNameLength {
		return ErrFeedNameTooLong
	}

	return nil
}

type URI struct {
	Type string
	Path string
}

func (u URI) IsZero() bool {
	return u.Type == "" && u.Path == ""
}

func (u URI) String() string {
	return fmt.Sprintf("%s://%s", u.Type, u.Path)
}

// TwtxtUserAgent ...
type TwtxtUserAgent interface {
	fmt.Stringer

	// IsPod returns true if the Twtxt client's User-Agent appears to be a Yarn.social pod (single or multi-user).
	IsPod() bool

	// PodBaseURL returns the base URL of the client's User-Agent if it appears to be a Yarn.social pod (single or multi-user).
	PodBaseURL() string

	// IsPublicURL returns true if the Twtxt client's User-Agent is from what appears to be the public internet.
	IsPublicURL() bool

	// Followers returns a list of followers for this client follows, in the case of a
	// single user agent, it is simply a list of itself, with a multi-user agent the
	// client (i.e: a `yarnd` pod) is aksed who followers the user/feed by requesting
	// the whoFollows resource
	Followers(conf *Config) types.Followers
}

// TwtxtUserAgent interface guards
var (
	_ TwtxtUserAgent = (*SingleUserAgent)(nil)
	_ TwtxtUserAgent = (*MultiUserAgent)(nil)
	_ TwtxtUserAgent = (*YarndUserAgent)(nil)
)

// twtxtUserAgent is a base class for both single and multi-user Twtxt User Agents.
type twtxtUserAgent struct {
	Client string
}

func (ua *twtxtUserAgent) IsPod() bool {
	return strings.HasPrefix(ua.Client, "yarnd/")
}

func (ua *twtxtUserAgent) podBaseURL(uri, relativeURLToTrim string) string {
	if !ua.IsPod() {
		return ""
	}

	u, err := url.Parse(uri)
	if err != nil {
		log.WithError(err).Warnf("error parsing User-Agent URL: %s", uri)
		return ""
	}

	// Throw away the trailing part of the URL to get the base URL for this
	// yarnd instance. It might serve from a subdirectory, so we cannot simply
	// cut off the complete path.
	rel, _ := url.Parse(relativeURLToTrim)
	return NormalizeURL(u.ResolveReference(rel).String())
}

func (ua *twtxtUserAgent) isPublicURL(uri, userAgent string) bool {
	u, err := url.Parse(uri)
	if err != nil {
		log.WithError(err).Warn("error parsing User-Agent URL")
		return false
	}

	ips, err := net.LookupIP(u.Hostname())
	if err != nil {
		log.WithError(err).Warn("error looking up User-Agent IP")
		return false
	}

	if len(ips) == 0 {
		log.Warnf("User-Agent lookup failed for %s or has no resolvable IP", userAgent)
		return false
	}

	ip := ips[0]

	// 0.0.0.0 or ::
	if ip.IsUnspecified() {
		return false
	}

	// Link-local / Loopback
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}

	return !ipip.IsPrivate(ip)
}

// SingleUserAgent is a single Twtxt User Agent whether it be `tt`, `jenny` or a single-user `yarnd` client.
type SingleUserAgent struct {
	twtxtUserAgent
	Nick string
	URI  string
}

func (ua *SingleUserAgent) String() string {
	// <client>/<version> (+<source.url>; @<source.nick>)
	return fmt.Sprintf("%s (+%s; @%s)", ua.Client, ua.URI, ua.Nick)
}

func (ua *SingleUserAgent) PodBaseURL() string {
	// get rid of the trailing '/user/foo/twtxt.txt'
	return ua.podBaseURL(ua.URI, "../..")
}

func (ua *SingleUserAgent) IsPublicURL() bool {
	return ua.isPublicURL(ua.URI, ua.String())
}

func (ua *SingleUserAgent) Followers(conf *Config) types.Followers {
	return types.Followers{
		&types.Follower{
			Nick:       ua.Nick,
			URI:        ua.URI,
			LastSeenAt: time.Now(),
		},
	}
}

// MultiUserAgent is a multi-user Twtxt client, currently only `yarnd` is such a client.
type MultiUserAgent struct {
	twtxtUserAgent
	WhoFollowsURL string
	SupportURL    string
}

func (ua *MultiUserAgent) String() string {
	// <client>/<version> (~<whoFollowsURL>; contact=<supportURL>)
	return fmt.Sprintf("%s (~%s; contact=%s)", ua.Client, ua.WhoFollowsURL, ua.SupportURL)
}

func (ua *MultiUserAgent) PodBaseURL() string {
	// get rid of the trailing '/whoFollows?followers=42&token=abc'
	return ua.podBaseURL(ua.WhoFollowsURL, "./")
}

func (ua *MultiUserAgent) IsPublicURL() bool {
	return ua.isPublicURL(ua.WhoFollowsURL, ua.String())
}

func (ua *MultiUserAgent) Followers(conf *Config) types.Followers {
	var followers types.Followers

	headers := make(http.Header)
	headers.Set("Accept", "application/json")

	res, err := RequestHTTP(conf, http.MethodGet, ua.WhoFollowsURL, headers)
	if err != nil {
		log.WithError(err).Errorf("error fetching whoFollows from %s", ua)
		return nil
	}
	defer res.Body.Close()

	if res.StatusCode/100 != 2 {
		log.Errorf("HTTP %s response for whoFollows resource from %s", res.Status, ua)
		return nil
	}

	if ctype := res.Header.Get("Content-Type"); ctype != "" {
		mediaType, _, err := mime.ParseMediaType(ctype)
		if err != nil {
			log.WithError(err).Errorf("error parsing content type header '%s' for whoFollows resoruce from %s", ctype, ua)
			return nil
		}
		if mediaType != "application/json" {
			log.Errorf("non-JSON response '%s' for whoFollows resource from %s", ctype, ua)
			return nil
		}
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		log.WithError(err).Errorf("error reading response body for whoFollows resource from %s", ua)
		return nil
	}

	kv := make(map[string]string)
	if err := json.Unmarshal(data, &kv); err != nil {
		// XXX: This only exists for backwards compatibility in 0.11.x where this got changed.
		// TODOL Remove post 0.12.x and adhere to the spec (map of nick -> uri)
		if err := json.Unmarshal(data, &followers); err == nil {
			return followers
		}
		log.WithError(err).Errorf("error deserializing whoFollows response from %s", ua)
		return nil
	}
	for k, v := range kv {
		followers = append(followers, &types.Follower{Nick: k, URI: v, LastSeenAt: time.Now()})
	}

	return followers
}

// YarndUserAgent is a generic `yarnd` client.
type YarndUserAgent struct {
	twtxtUserAgent
	Name       string
	SupportURL string
}

func (ua *YarndUserAgent) String() string {
	// <client>/<version> (Pod: <name> Support: <supportURL>)
	return fmt.Sprintf("%s (Pod: %s Support: %s)", ua.Client, ua.Name, ua.SupportURL)
}

func (ua *YarndUserAgent) PodBaseURL() string {
	// get rid of the trailing '/support'
	return ua.podBaseURL(ua.SupportURL, "./")
}

func (ua *YarndUserAgent) IsPublicURL() bool {
	return ua.isPublicURL(ua.SupportURL, ua.String())
}

func (ua *YarndUserAgent) Followers(conf *Config) types.Followers {
	return nil
}

func ParseUserAgent(ua string) (TwtxtUserAgent, error) {
	if match := singleUserUARegex.FindStringSubmatch(ua); match != nil {
		return &SingleUserAgent{
			twtxtUserAgent: twtxtUserAgent{Client: match[1]},
			URI:            match[2],
			Nick:           match[3],
		}, nil
	}

	if match := multiUserUARegex.FindStringSubmatch(ua); match != nil {
		return &MultiUserAgent{
			twtxtUserAgent: twtxtUserAgent{Client: match[1]},
			WhoFollowsURL:  match[2],
			SupportURL:     match[3],
		}, nil
	}

	if match := yarndUserUARegex.FindStringSubmatch(ua); match != nil {
		return &YarndUserAgent{
			twtxtUserAgent: twtxtUserAgent{Client: match[1]},
			Name:           match[2],
			SupportURL:     match[3],
		}, nil
	}

	return nil, ErrInvalidUserAgent
}

func ParseURI(uri string) (*URI, error) {
	parts := strings.Split(uri, "://")
	if len(parts) == 2 {
		return &URI{Type: strings.ToLower(parts[0]), Path: parts[1]}, nil
	}
	return nil, fmt.Errorf("invalid uri: %s", uri)
}

func NormalizeUsername(username string) string {
	return strings.TrimSpace(strings.ToLower(username))
}

func NormalizeURL(url string) string {
	if url == "" {
		return ""
	}

	u, err := urlx.Parse(url)
	if err != nil {
		log.WithError(err).Errorf("NormalizeURL: error parsing url %s", url)
		return ""
	}
	switch u.Scheme {
	case "http":
		u.Host = strings.TrimSuffix(u.Host, ":80")
	case "https":
		u.Host = strings.TrimSuffix(u.Host, ":443")
	}
	u.User = nil
	u.Fragment = ""
	u.Path = strings.TrimSuffix(u.Path, "/")
	norm, err := urlx.Normalize(u)
	if err != nil {
		log.WithError(err).Errorf("error normalizing url %s", url)
		return ""
	}
	return norm
}

// hostPath strips scheme, default ports, and trailing slash,
// then returns host+path[+?query] for comparison.
func hostPath(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		// fallback to naive strip if URL is malformed
		return strings.TrimPrefix(raw, func() string {
			if i := strings.Index(raw, "://"); i >= 0 {
				return raw[:i+3]
			}
			return ""
		}())
	}

	// lower-case the host, remove default port if present
	host := strings.ToLower(u.Host)
	if h, port, _ := net.SplitHostPort(host); port != "" {
		if (u.Scheme == "http" && port == "80") || (u.Scheme == "https" && port == "443") {
			host = h
		}
	}

	// clean and trim any trailing slash from the path
	path := strings.TrimSuffix(u.EscapedPath(), "/")

	// include query if you consider ?foo=bar part of uniqueness:
	if u.RawQuery != "" {
		path += "?" + u.RawQuery
	}

	return host + path
}

// RedirectRefererURL constructs a Redirect URL from the given Request URL
// and possibly Referer, if the Referer's Base URL matches the Pod's Base URL
// will return the Referer URL otherwise the defaultURL. This is primarily used
// to redirect a user from a successful /login back to the page they were on.
func RedirectRefererURL(r *http.Request, conf *Config, defaultURL string) string {
	referer := NormalizeURL(r.Header.Get("Referer"))
	if referer != "" && strings.HasPrefix(referer, conf.BaseURL) {
		refererURL, err := url.Parse(referer)
		if err != nil {
			log.WithError(err).Warnf("error parsing referer url")
			return defaultURL
		}
		// Get rid of the Bookmarklet (if any)
		// (but only if it's not the /external profile view)
		if !strings.HasPrefix(strings.TrimPrefix(refererURL.Path, "/"), "external") {
			refererURL.RawQuery = ""
			refererURL.RawFragment = ""
		}
		return refererURL.String()
	}

	return defaultURL
}

func HostnameFromURL(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		log.WithError(err).Warnf("HostnameFromURL(): error parsing url: %s", uri)
		return uri
	}

	return u.Hostname()
}

func BaseFromURL(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		log.WithError(err).Warnf("BaseFromURL(): error parsing url: %s", uri)
		return uri
	}

	u.Fragment = ""
	u.RawFragment = ""
	u.Path = ""
	u.RawPath = ""
	u.RawQuery = ""

	return u.String()
}

func PrettyURL(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		log.WithError(err).Warnf("PrettyURL(): error parsing url: %s", uri)
		return uri
	}

	return fmt.Sprintf("%s/%s", u.Hostname(), strings.TrimPrefix(u.EscapedPath(), "/"))
}

// IsAdminUserFactory returns a function that returns true if the user provided
// is the configured pod administrator, false otherwise.
func IsAdminUserFactory(conf *Config) func(user *User) bool {
	return func(user *User) bool {
		return NormalizeUsername(conf.AdminUser) == NormalizeUsername(user.Username)
	}
}

func UserURL(url string) string {
	if strings.HasSuffix(url, "/twtxt.txt") {
		return strings.TrimSuffix(url, "twtxt.txt")
	}
	return url
}

func URLForMedia(baseURL, name string) string {
	return fmt.Sprintf(
		"%s/media/%s",
		strings.TrimSuffix(baseURL, "/"),
		name,
	)
}

func URLForPage(baseURL, page string) string {
	return fmt.Sprintf(
		"%s/%s",
		strings.TrimSuffix(baseURL, "/"),
		page,
	)
}

func URLForTwt(baseURL, hash string) string {
	return fmt.Sprintf(
		"%s/twt/%s",
		strings.TrimSuffix(baseURL, "/"),
		hash,
	)
}

func URLForUser(baseURL, username string) string {
	return fmt.Sprintf(
		"%s/user/%s/twtxt.txt",
		strings.TrimSuffix(baseURL, "/"),
		username,
	)
}

func URLForFollowing(baseURL, username string) string {
	return fmt.Sprintf(
		"%s/user/%s/following",
		strings.TrimSuffix(baseURL, "/"),
		username,
	)
}

func URLForFollowers(baseURL, username string) string {
	return fmt.Sprintf(
		"%s/user/%s/followers",
		strings.TrimSuffix(baseURL, "/"),
		username,
	)
}

func URLForAvatar(baseURL, username, avatarHash string) string {
	uri := fmt.Sprintf(
		"%s/user/%s/avatar",
		strings.TrimSuffix(baseURL, "/"),
		username,
	)
	if avatarHash != "" {
		uri += "#" + avatarHash
	}
	return uri
}

func URLForExternalProfile(conf *Config, uri string) string {
	return fmt.Sprintf(
		"%s/external?uri=%s",
		strings.TrimSuffix(conf.BaseURL, "/"),
		uri,
	)
}

func URLForExternalAvatar(conf *Config, uri string) string {
	return fmt.Sprintf(
		"%s/externalAvatar?uri=%s",
		strings.TrimSuffix(conf.BaseURL, "/"),
		uri,
	)
}

// GetConvLength returns the number of twts in a conv.
func GetConvLength(_ *Config, cache Cacher) func(twt types.Twt, u *User) int {
	return func(twt types.Twt, u *User) int {
		opts := &QueryOptions{
			Limit:   1,
			Exclude: u.MutedList(),
		}
		subject := twt.Subject().Text()
		_, total, err := cache.GetBySubject(subject, opts)
		if err != nil {
			log.WithError(err).Warnf("error fetching conv length for %s", subject)
			return 0
		}
		return total
	}
}

// GetForkLength returns the number of twts in a fork.
func GetForkLength(_ *Config, cache Cacher) func(twt types.Twt, u *User) int {
	return func(twt types.Twt, u *User) int {
		opts := &QueryOptions{
			Limit:   1,
			Exclude: u.MutedList(),
		}
		subject := "#" + twt.Hash()
		_, total, err := cache.GetBySubject(subject, opts)
		if err != nil {
			log.WithError(err).Warnf("error fetching fork length for %s", subject)
			return 0
		}
		return total
	}
}

// GetFeedTypeClass returns a function that returns the class name for a given feed URI. The class name is used to style the feed in the UI. The function returns an empty string if the feed type is not recognized. The function is case-insensitive. The function is also memoized to avoid redundant calculations. The function is also cached to avoid redundant calculations. The function
func GetFeedTypeClass(conf *Config, cache Cacher) func(uri string) string {
	return func(uri string) string {
		var metadata url.Values
		if cachedTwter := cache.GetTwter(uri); cachedTwter != nil {
			metadata = cachedTwter.Metadata
		}
		switch types.GetFeedType(uri, metadata) {
		case types.FeedTypeBot:
			return "robot"
		case types.FeedTypeRSS:
			return "rss"
		default:
			// User, person, human, etc defaults to no icon.
			return ""
		}
	}
}

func ExtractHashFromSubject(subject string) string {
	// TODO: Pre-compile this regex pattern?
	re := regexp.MustCompile(`#([a-z0-9]+)`)
	if match := re.FindStringSubmatch(subject); match != nil {
		return match[1]
	}

	return ""
}

func GetLookupMatches(conf *Config, nick string, uri string) (avatar, domain string) {
	isLocalURL := IsLocalURLFactory(conf)

	if isLocalURL(uri) {
		avatar = URLForAvatar(conf.BaseURL, nick, "")
		re := regexp.MustCompile(`https?:\/\/(.+?)\/user\/`)
		domain = re.FindStringSubmatch(strings.ToLower(avatar))[1]
	} else {
		avatar = URLForExternalAvatar(conf, uri)
		re := regexp.MustCompile(`uri=https?:\/\/(.+?)\/user\/`)
		if matches := re.FindStringSubmatch(strings.ToLower(avatar)); matches != nil {
			domain = matches[1]
		}
	}

	return
}

func GetTwtConvSubjectHash(cache Cacher, twt types.Twt) (string, string) {
	subject := twt.Subject().Text()
	if subject == "" {
		return "", ""
	}

	hash := ExtractHashFromSubject(subject)
	if _, ok := cache.Lookup(hash, nil); !ok {
		return "", ""
	}

	return "#" + hash, hash
}

func URLForConvFactory(conf *Config, cache Cacher) func(twt types.Twt) string {
	return func(twt types.Twt) string {
		if _, hash := GetTwtConvSubjectHash(cache, twt); hash != "" {
			return fmt.Sprintf(
				"%s/conv/%s",
				strings.TrimSuffix(conf.BaseURL, "/"),
				hash,
			)
		}
		return ""
	}
}

func URLForForkFactory(conf *Config, cache Cacher) func(twt types.Twt) string {
	return func(twt types.Twt) string {
		return fmt.Sprintf(
			"%s/conv/%s",
			strings.TrimSuffix(conf.BaseURL, "/"),
			twt.Hash(),
		)
	}
}

func DivMod(n, d int) (q, r int) {
	q = n / d
	r = n % d
	return
}

func URLForRootConvFactory(conf *Config, cache Cacher, withPager bool) func(twt types.Twt, user *User) string {
	// getTwtsInConv loads all tweets in a conversation for the given hash.
	// It uses a large limit (conf.TwtsPerPage * maxPagesToPull) and offset = 0.
	getTwtsInConv := func(hash string, _ *User) types.Twts {
		twt, inCache := cache.Lookup(hash, nil)
		if twt.IsZero() {
			return nil
		}

		// Use a "full load" limit for conversation threads.
		fullLimit := conf.TwtsPerPage * 3
		twts, _, err := cache.GetBySubject("#"+hash, &QueryOptions{Limit: fullLimit})
		if err != nil {
			log.WithError(err).Error("error loading conversation twts")
			return nil
		}

		// If the original tweet wasn't in cache, add it.
		if !inCache {
			twts = append(twts, twt)
		}

		// Sort in reverse order (most recent first).
		sort.Sort(sort.Reverse(twts))

		if len(twts) == 0 {
			return nil
		}

		return twts
	}

	// findTwtInTwts searches for the given twt in a slice and returns its index.
	findTwtInTwts := func(twt types.Twt, twts types.Twts) int {
		for i, t := range twts {
			if t.Hash() == twt.Hash() {
				return i
			}
		}
		return 0
	}

	return func(twt types.Twt, user *User) string {
		// Get the conversation subject hash.
		_, hash := GetTwtConvSubjectHash(cache, twt)
		if hash != "" && hash != twt.Hash() {
			page := 1
			// Only compute the page if withPager is true.
			if withPager {
				twts := getTwtsInConv(hash, user)
				if len(twts) > 0 {
					n := findTwtInTwts(twt, twts)
					// DivMod returns the quotient and remainder.
					q, r := DivMod(n, conf.TwtsPerPage)
					if r > 0 {
						page = q + 1
					} else {
						page = q
					}
				}
			}
			return fmt.Sprintf(
				"%s/conv/%s?p=%d;#%s",
				strings.TrimSuffix(conf.BaseURL, "/"),
				hash, page, twt.Hash(),
			)
		}
		return ""
	}
}

func URLForTag(baseURL, tag string) string {
	return fmt.Sprintf(
		"%s/search?q=%%23%s",
		strings.TrimSuffix(baseURL, "/"),
		tag,
	)
}

func URLForTask(baseURL, uuid string) string {
	return fmt.Sprintf(
		"%s/task/%s",
		strings.TrimSuffix(baseURL, "/"),
		uuid,
	)
}

func URLForWhoFollows(baseURL string, feed FetchFeedRequest, feedFollowers int) string {
	return fmt.Sprintf(
		"%s/whoFollows?followers=%d&token=%s",
		strings.TrimSuffix(baseURL, "/"),
		// Include the number of followers, so feed owners can use this as a vague
		// indicator to avoid refetching our Who Follows Resource if the number did
		// not change since they last checked their followers.
		feedFollowers,
		GenerateWhoFollowsToken(feed.URI),
	)
}

// SafeParseInt ...
func SafeParseInt(s string, d int) int {
	n, e := strconv.Atoi(s)
	if e != nil {
		return d
	}
	return n
}

// ValidateUsername validates the username before allowing it to be created.
// This ensures usernames match a defined pattern and that some usernames
// that are reserved are never used by users.
func ValidateUsername(username string) error {
	username = NormalizeUsername(username)

	if !validUsername.MatchString(username) {
		return ErrInvalidUsername
	}

	for _, reservedUsername := range reservedUsernames {
		if username == reservedUsername {
			return ErrReservedUsername
		}
	}

	if len(username) > maxUsernameLength {
		return ErrUsernameTooLong
	}

	return nil
}

// UnparseTwtFactory is the opposite of CleanTwt and ExpandMentions/ExpandTags
func UnparseTwtFactory(conf *Config, cache Cacher) func(twt types.Twt) string {
	isLocalURL := IsLocalURLFactory(conf)
	return func(twt types.Twt) string {
		text := fmt.Sprintf("%l", twt)
		//XXX: Why is this treating the subject as a tag and encoding the subject
		//XXX: as #<bash http://example.com/search?q=<hash>
		//XXX: wtf?!
		//text := twt.FormatText(types.LiteralFmt, conf)

		text = strings.ReplaceAll(text, "\u2028", "\n")

		if subject, _ := GetTwtConvSubjectHash(cache, twt); subject != "" {
			text = strings.ReplaceAll(text, "("+subject+")", "")
			text = strings.TrimSpace(text)
		}

		re := regexp.MustCompile(`(@|#)<([^ ]+) *([^>]+)>`)
		return re.ReplaceAllStringFunc(text, func(match string) string {
			parts := re.FindStringSubmatch(match)
			prefix, nick, uri := parts[1], parts[2], parts[3]

			switch prefix {
			case "@":
				if uri != "" && !isLocalURL(uri) {
					u, err := url.Parse(uri)
					if err != nil {
						log.WithField("uri", uri).Warn("UnparseTwt(): error parsing uri")
						return match
					}
					return fmt.Sprintf("@%s@%s", nick, u.Hostname())
				}
				return fmt.Sprintf("@%s", nick)
			case "#":
				return fmt.Sprintf("#%s", nick)
			default:
				log.
					WithField("prefix", prefix).
					WithField("nick", nick).
					WithField("uri", uri).
					Warn("UnprocessTwt(): invalid prefix")
			}
			return match
		})
	}
}

type FilterTwtsFunc func(user *User, twts types.Twts) types.Twts

// CleanTwt cleans a twt's text, replacing new lines with spaces and
// stripping surrounding spaces.
func CleanTwt(text string) string {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\n", "\u2028")
	return text
}

// RenderAudio ...
func RenderAudio(conf *Config, uri, title, renderAs string, full bool) string {
	// XXX: `renderAs` is ignored for Audio right now
	// XXX: `full` is ignored for Audio right now

	isLocalURL := IsLocalURLFactory(conf)

	if isLocalURL(uri) {
		u, err := url.Parse(uri)
		if err != nil {
			log.WithError(err).Warnf("error parsing uri: %s", uri)
			return ""
		}

		mp3URI := u.String()

		return fmt.Sprintf(`<audio controls="controls" title="%s">
  <source type="audio/mp3" src="%s"></source>
  Your browser does not support the audio element.
</audio>`, title, mp3URI)
	}

	return fmt.Sprintf(`<audio controls="controls" title="%s">
  <source type="audio/mp3" src="%s"></source>
  Your browser does not support the audio element.
</audio>`, title, uri)
}

// RenderImage ...
func RenderImage(user *User, conf *Config, uri, caption, alt, renderAs string, full bool) string {
	// isMediaURI tries to guess whether the `uri` looks like it _might_ have come from another `yarnd` pod
	// if so we _assume_ we can optionally download original quality media from it by appending ?full=1

	var (
		uuid       string
		isMediaURI bool
	)

	u, err := url.Parse(uri)
	if err != nil {
		log.WithError(err).Warnf("error parsing uri: %s", uri)
		return ""
	}

	if mediaURIRegex.MatchString(u.Path) {
		isMediaURI = true
		uuid = strings.TrimSuffix(strings.Split(u.Path, "/")[2], ".png")
	} else {
		uuid = u.Path
	}

	if alt != "" {
		alt = ` alt="` + alt + `"`
	}

	if renderAs == "" || renderAs == "inline" {
		if isMediaURI && full {
			return fmt.Sprintf(`<img loading=lazy src="%s?full=1" title="%s"%s/>`, uri, caption, alt)
			// return fmt.Sprintf(`<a href="%s?full=1" target="_blank"><img loading=lazy src="%s?full=1" title="%s"%s/></a>`, uri, uri, caption, alt)

		}
		return fmt.Sprintf(`<img loading=lazy src="%s" title="%s"%s/>`, uri, caption, alt)
		// return fmt.Sprintf(`<a href="%s?full=1" target="_blank"><img loading=lazy src="%s" title="%s"%s/>`, uri, uri, caption, alt)
	}

	isLocalURL := IsLocalURLFactory(conf)

	imgURI := u.String()
	if isMediaURI && full {
		imgURI += "?full=1"
	}
	fullImgURI := u.String()
	if isMediaURI {
		fullImgURI += "?full=1"
	}

	title := "Open to view original quality"

	if !isLocalURL(uri) {
		title = fmt.Sprintf(
			`%s on %s`,
			title, u.Hostname(),
		)
	}

	// XXX: Captions are disabled for lightbox preview if readmore is enabled.
	isCaption := ""
	if !user.VisibilityReadmore && caption != "" {
		isCaption = fmt.Sprintf(
			`<lightbox class="caption" data-target="%s">%s</lightbox>`,
			uuid, caption,
		)
	}

	return fmt.Sprintf(
		`<lightbox class="center-cropped caption-wrap">
			 <a class="img-orig-open" href="%s" title="%s"%s target="_blank" _="on click call toggleModal(event)">%s
				 <img loading=lazy src="%s" data-target="%s" />
			 </a>
		 </lightbox>
		 <dialog id="%s">
        <figure>
          <img loading=lazy src="%s" />
          <figcaption><a class="img-dl" href="%s" hx-boost="false">Download</a><p>%s</p></figcaption>
        </figure>
      </dialog>`,
		imgURI, title, alt, isCaption, imgURI, uuid, uuid, fullImgURI, fullImgURI, caption,
	)
}

// RenderVideo ...
func RenderVideo(conf *Config, uri, title, renderAs string, full bool) string {
	// XXX: `renderAs` is ignored for Video right now
	// XXX: `full` is ignored for for Video right now

	isLocalURL := IsLocalURLFactory(conf)

	if isLocalURL(uri) {
		u, err := url.Parse(uri)
		if err != nil {
			log.WithError(err).Warnf("error parsing uri: %s", uri)
			return ""
		}

		u.Path = ReplaceExt(u.Path, "")
		posterURI := u.String()

		return fmt.Sprintf(`<video controls playsinline preload="auto" title="%s" poster="%s">
    <source type="video/mp4" src="%s" />
    Your browser does not support the video element.
  </video>`, title, posterURI, uri)
	}

	return fmt.Sprintf(`<video controls playsinline preload="auto" title="%s">
    <source type="video/mp4" src="%s" />
    Your browser does not support the video element.
    </video>`, title, uri)
}

// PreprocessMedia ...
func PreprocessMedia(user *User, conf *Config, u *url.URL, title, alt, renderAs string, display, full bool) string {
	var html string

	// Normalize the domain name
	domain := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	permitted, local := conf.PermittedImage(domain)

	if permitted && display {
		if local {
			// Ensure all local links match our BaseURL scheme
			u.Scheme = conf.baseURL.Scheme
		} else {
			// Ensure all extern links are served over TLS
			u.Scheme = "https"
		}

		switch filepath.Ext(u.Path) {
		case ".mp4":
			html = RenderVideo(conf, u.String(), title, renderAs, full)
		case ".mp3":
			html = RenderAudio(conf, u.String(), title, renderAs, full)
		default:
			html = RenderImage(user, conf, u.String(), title, alt, renderAs, full)
		}
	} else {
		var mtype, mtypeIcon string
		switch filepath.Ext(u.Path) {
		case ".mp4":
			mtype = "Video"
			mtypeIcon = "ti-movie"
		case ".mp3":
			mtype = "Audio"
			mtypeIcon = "ti-device-speaker"
		default:
			mtype = "Image"
			mtypeIcon = "ti-photo"
		}

		if alt != "" {
			alt = ` alt="` + alt + `"`
		}

		if full {
			html = fmt.Sprintf(
				`<p><a class="e-media" href="%s?full=1" title="%s"%s target="_blank"><i class="ti %s"></i> %s</a></p>`,
				u.String(), title, alt, mtypeIcon, mtype,
			)
		} else {
			html = fmt.Sprintf(
				`<p><a class="e-media" href="%s" title="%s"%s target="_blank"><i class="ti %s"></i> %s</a></p>`,
				u.String(), title, alt, mtypeIcon, mtype,
			)
		}
	}

	return html
}

func FormatForDateTime(t time.Time, timeFormat string) string {
	dateTimeFormat := ""

	if timeFormat == "" {
		timeFormat = "3:04PM"
	}

	dt := time.Since(t)

	if dt > YearAgo {
		dateTimeFormat = "Mon, Jan 2 %s 2006"
	} else if dt > MonthAgo {
		dateTimeFormat = "Mon, Jan 2 %s"
	} else if dt > WeekAgo {
		dateTimeFormat = "Mon, Jan 2 %s"
	} else if dt > DayAgo {
		dateTimeFormat = "Mon 2, %s"
	} else {
		dateTimeFormat = "%s"
	}

	return fmt.Sprintf(dateTimeFormat, timeFormat)
}

type URLProcessor struct {
	conf   *Config
	user   *User
	target string

	Images []string
}

type EmbedRule struct {
	Pattern string         `json:"pattern"`
	pattern *regexp.Regexp `json:"-"`
	Source  string         `json:"src"`
	Class   string         `json:"class"`
	Allow   string         `json:"allow"`
}

func (up *URLProcessor) RenderNodeHook(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
	// renderAs (one of inline or lightbox)
	var renderAs string

	if (up.user != nil && up.user.DisplayImagesPreference == "gallery") || (up.user == nil && up.conf.DisplayImagesPreference == "gallery") {
		renderAs = "lightbox"
	} else if (up.user != nil && up.user.DisplayImagesPreference == "lightbox") || (up.user == nil && up.conf.DisplayImagesPreference == "lightbox") {
		renderAs = "lightbox"
	} else if (up.user != nil && up.user.DisplayImagesPreference == "inline") || (up.user == nil && up.conf.DisplayImagesPreference == "inline") {
		renderAs = "inline"
	} else {
		renderAs = "inline"
	}

	display := up.conf.DisplayMedia
	if up.user != nil {
		display = up.user.DisplayMedia
	}

	full := up.conf.OriginalMedia
	if up.user != nil {
		full = up.user.OriginalMedia
	}

	// Converts links to embed elements.
	link, ok := node.(*ast.Link)
	if ok && entering && display {
		dst := string(link.Destination)
		title := string(link.Title)

		for _, rule := range up.conf.embedRules {
			if !rule.pattern.MatchString(dst) {
				continue
			}

			for _, child := range link.Container.GetChildren() {
				text, ok := child.(*ast.Text)
				if !ok {
					continue
				}

				literal := string(text.Literal)

				class := "embed-link"
				if literal == dst {
					class += " embed-link-is-url"
				}

				html := fmt.Sprintf(
					`<a class="%s" href="%s" target="%s" rel="nofollow">%s</a>`,
					class,
					dst,
					up.target,
					literal,
				)

				_, _ = io.WriteString(w, html)
				break
			}

			html := fmt.Sprintf(`
				</p>
					<iframe
						loading="lazy"
						frameborder="0"
						src="%s"
						class="%s"
						allow="%s"
						title="%s">
					</iframe>
				<p>
			`, rule.pattern.ReplaceAllString(dst, rule.Source), rule.Class, rule.Allow, title)

			_, _ = io.WriteString(w, html)
			return ast.SkipChildren, true
		}
	}

	// Ensure only permitted ![](url) images
	image, ok := node.(*ast.Image)
	if ok && entering {
		u, err := url.Parse(string(image.Destination))
		if err != nil {
			log.WithError(err).Warn("TwtFactory: error parsing url")
			return ast.GoToNext, false
		}

		alt := string(image.Title)
		if children := image.Container.GetChildren(); len(children) > 0 {
			for _, c := range children {
				if txt, ok := c.(*ast.Text); ok {
					alt = string(txt.Literal)
				}
			}
		}

		html := PreprocessMedia(up.user, up.conf, u, string(image.Title), alt, renderAs, display, full)
		if _, ok := node.GetParent().(*ast.Paragraph); ok && renderAs != "inline" {
			html = fmt.Sprintf("</p>%s<p>", html)
		}
		_, _ = io.WriteString(w, html)

		return ast.SkipChildren, true
	}

	span, ok := node.(*ast.HTMLSpan)
	if !ok {
		return ast.GoToNext, false
	}

	leaf := span.Leaf
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(leaf.Literal))
	if err != nil {
		log.WithError(err).Warn("error parsing HTMLSpan")
		return ast.GoToNext, false
	}

	// Ensure only permitted img src=(s) and fix non-secure links
	img := doc.Find("img")
	if img.Length() > 0 {
		src, ok := img.Attr("src")
		if !ok {
			return ast.GoToNext, false
		}

		alt, _ := img.Attr("alt")
		title, _ := img.Attr("title")

		u, err := url.Parse(src)
		if err != nil {
			log.WithError(err).Warn("error parsing URL")
			return ast.GoToNext, false
		}

		html := PreprocessMedia(up.user, up.conf, u, title, alt, renderAs, display, full)
		if _, ok := node.GetParent().(*ast.Paragraph); ok && renderAs != "inline" {
			html = fmt.Sprintf("</p>%s<p>", html)
		}
		_, _ = io.WriteString(w, html)

		return ast.GoToNext, true
	}

	// Ensure that User.OpenLinksInPreference is respected
	a := doc.Find("a")
	if a.Length() > 0 {
		html := fmt.Sprintf(
			`<a href="%s" target="%s" rel="nofollow">`,
			a.AttrOr("href", "#"), up.target,
		)
		_, _ = io.WriteString(w, html)
		return ast.GoToNext, true
	}

	// Let it go! Let it go!
	return ast.GoToNext, false
}

var _ types.FmtOpts = (*nickForURLFmtOpts)(nil)
var _ lextwt.NickForURLResolver = (*nickForURLFmtOpts)(nil)

// nickForURLFmtOpts implements both the standard types.FmtOpts and additional
// lextwt.NickForURLResolver interfaces for HTML rendering of twts.
type nickForURLFmtOpts struct {
	*Config
	cache Cacher
}

// NickForURL looks up the nickname for URL-only mentions in the form of
// '@<url>'. When unknown, an empty string is returned.
func (opts *nickForURLFmtOpts) NickForURL(uri string) string {
	// TODO: Refactor this?
	/* FIXME: Fix this
	if memoryCache, ok := opts.cache.(Cacher); ok {
		return memoryCache.NickForURL(uri)
	}
	*/

	twter := opts.cache.GetTwter(uri)
	if twter == nil {
		return ""
	}
	return twter.Nick
}

// FormatTwtFactory formats a twt into a valid HTML snippet
func FormatTwtFactory(conf *Config, cache Cacher) func(types.Twt, types.TwtTextFormat, *User) template.HTML {
	return func(twt types.Twt, format types.TwtTextFormat, user *User) template.HTML {
		extensions := parser.NoIntraEmphasis | parser.FencedCode |
			parser.Autolink | parser.Strikethrough | parser.SpaceHeadings |
			parser.NoEmptyLineBeforeBlock | parser.HardLineBreak

		mdParser := parser.NewWithExtensions(extensions)

		htmlFlags := html.Smartypants | html.SmartypantsDashes | html.SmartypantsLatexDashes

		openLinksIn := conf.OpenLinksInPreference
		if user != nil {
			openLinksIn = user.OpenLinksInPreference
		}

		var target string
		if strings.ToLower(openLinksIn) == "newwindow" {
			target = "_blank"
			htmlFlags = htmlFlags | html.HrefTargetBlank
		} else {
			target = "_self"
		}

		up := &URLProcessor{conf: conf, user: user, target: target}

		opts := html.RendererOptions{
			Flags:          htmlFlags,
			Generator:      "",
			RenderNodeHook: up.RenderNodeHook,
		}

		renderer := html.NewRenderer(opts)

		// copy alt to title if present.
		if cp, ok := twt.(*lextwt.Twt); ok {
			twt = cp.Clone()
			for _, m := range twt.Links() {
				if link, ok := m.(*lextwt.Link); ok {
					link.TextToTitle()
				}
			}
		}

		// XXX: Note that even through we're calling twt.FormatText(types.HTMLFmt, conf) here
		// the output is in fact probably (mostly) Markdown anyway, we're just asking the lexttwt Parser
		// to render nodes as HTML snippets (like @-mentions).
		markdownInput := twt.FormatText(format, &nickForURLFmtOpts{conf, cache})
		if subject, _ := GetTwtConvSubjectHash(cache, twt); subject != "" {
			markdownInput = strings.ReplaceAll(markdownInput, subject, "")
			markdownInput = strings.TrimSpace(markdownInput)
		}

		md := []byte(markdownInput)
		maybeUnsafeHTML := markdown.ToHTML(md, mdParser, renderer)

		p := bluemonday.StrictPolicy()
		p.AllowStandardURLs()
		// Override the allowed schemas as p.AllowStandardURLs only permits http, https and mailto
		p.AllowURLSchemes("mailto", "http", "https", "gemini", "gopher")
		p.AllowElements("a", "span", "img", "strong", "em", "del", "p", "br", "blockquote", "ul", "ol", "li", "pre", "code", "figure", "figcaption")
		p.AllowAttrs("_", "hx-boost").OnElements("a")
		p.AllowAttrs("href").OnElements("a")
		p.AllowAttrs("src").OnElements("img")
		p.AllowAttrs("id").OnElements("dialog")
		p.AllowAttrs("id", "controls").OnElements("audio")
		p.AllowAttrs("id", "controls", "playsinline", "preload", "poster").OnElements("video")
		p.AllowAttrs("src", "type").OnElements("source")
		p.AllowAttrs("aria-label", "class", "data-target", "target").OnElements("a")
		p.AllowAttrs("class", "data-target").OnElements("i", "lightbox")
		p.AllowAttrs("alt", "title", "loading", "data-target", "data-tooltip").OnElements("a", "img")

		p.AllowElements("iframe")
		p.AllowAttrs("loading").Matching(regexp.MustCompile(`^lazy$`)).OnElements("iframe")
		p.AllowAttrs("frameborder").Matching(bluemonday.Number).OnElements("iframe")
		p.AllowAttrs("class").OnElements("iframe")
		p.AllowAttrs("title").OnElements("iframe")
		p.AllowAttrs("allow").Matching(regexp.MustCompile(`[a-z; -]*`)).OnElements("iframe")
		for _, rewrite := range conf.embedRules {
			replacement := regexp.MustCompile(`\\\$[0-9]+`)

			source := regexp.QuoteMeta(rewrite.Source)
			source = replacement.ReplaceAllString(source, ".*")

			source_pattern, err := regexp.Compile(source)
			if err != nil {
				continue
			}

			p.AllowAttrs("src").Matching(source_pattern).OnElements("iframe")
		}

		html := p.SanitizeBytes(maybeUnsafeHTML)

		return template.HTML(fmt.Sprintf(`<p>%s</p>`, html))
	}
}

func GetRootTwtFactory(conf *Config, cache Cacher) func(twt types.Twt, u *User) types.Twt {
	return func(twt types.Twt, u *User) types.Twt {
		_, hash := GetTwtConvSubjectHash(cache, twt)
		if hash == "" {
			return types.NilTwt
		}

		var rootTwt types.Twt

		if twt, inCache := cache.Lookup(hash, nil); inCache {
			rootTwt = twt
		} else {
			log.Warnf("unable to get context for twt: %s", hash)
			return types.NilTwt
		}

		if u.HasMuted(rootTwt.Twter().URI) {
			return types.NilTwt
		}

		return rootTwt
	}
}

// FormatTwtContextFactory formats a twt's context into a valid HTML snippet
// A Twt's Context is defined as the content of the Root Twt of the Conversation
// rendered in plain text up to a maximu length with an elipsis if longer...
func FormatTwtContextFactory(conf *Config, cache Cacher) func(twt types.Twt, u *User) template.HTML {
	getRootTwt := GetRootTwtFactory(conf, cache)
	return func(twt types.Twt, user *User) template.HTML {
		rootTwt := getRootTwt(twt, user)
		if rootTwt.IsZero() {
			return template.HTML("")
		}

		renderNodeHookFactory := func(target string) html.RenderNodeFunc {
			return func(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
				// Ensure only permitted ![](url) images
				image, ok := node.(*ast.Image)
				if ok && entering {
					u, err := url.Parse(string(image.Destination))
					if err != nil {
						log.WithError(err).Warn("TwtFactory: error parsing url")
						return ast.GoToNext, false
					}

					alt := string(image.Title)
					if children := image.Container.GetChildren(); len(children) > 0 {
						for _, c := range children {
							if txt, ok := c.(*ast.Text); ok {
								alt = string(txt.Literal)
							}
						}
					}
					if alt != "" {
						alt = ` alt="` + alt + `"`
					}

					src := u.String()
					html := fmt.Sprintf(
						`<a href="%s" title="%s"%s target="_blank"><i class="ti ti-external-link"></i> Media</a>`,
						src, image.Title, alt,
					)

					_, _ = io.WriteString(w, html)

					return ast.SkipChildren, true
				}

				span, ok := node.(*ast.HTMLSpan)
				if !ok {
					return ast.GoToNext, false
				}

				leaf := span.Leaf
				doc, err := goquery.NewDocumentFromReader(bytes.NewReader(leaf.Literal))
				if err != nil {
					log.WithError(err).Warn("error parsing HTMLSpan")
					return ast.GoToNext, false
				}

				// Ensure only permitted img src=(s) and fix non-secure links
				img := doc.Find("img")
				if img.Length() > 0 {
					src, ok := img.Attr("src")
					if !ok {
						return ast.GoToNext, false
					}

					alt, _ := img.Attr("alt")

					u, err := url.Parse(src)
					if err != nil {
						log.WithError(err).Warn("error parsing URL")
						return ast.GoToNext, false
					}

					html := fmt.Sprintf(
						`<a href="%s" alt="%s" target="%s"><i class="ti ti-external-link"></i> Media</a>`,
						u, alt, target,
					)

					_, _ = io.WriteString(w, html)

					return ast.GoToNext, true
				}

				// Ensure that User.OpenLinksInPreference is respected
				a := doc.Find("a")
				if a.Length() > 0 {
					html := fmt.Sprintf(
						`<a href="%s" target="%s" rel="nofollow">`,
						a.AttrOr("href", "#"), target,
					)
					_, _ = io.WriteString(w, html)
					return ast.GoToNext, true
				}

				// Let it go! Let it go!
				return ast.GoToNext, false
			}
		}

		extensions := parser.NoExtensions
		mdParser := parser.NewWithExtensions(extensions)
		htmlFlags := html.FlagsNone

		openLinksIn := conf.OpenLinksInPreference
		if user != nil {
			openLinksIn = user.OpenLinksInPreference
		}

		var target string
		if strings.ToLower(openLinksIn) == "newwindow" {
			target = "_blank"
			htmlFlags = htmlFlags | html.HrefTargetBlank
		} else {
			target = "_self"
		}

		opts := html.RendererOptions{
			Flags:          htmlFlags,
			Generator:      "",
			RenderNodeHook: renderNodeHookFactory(target),
		}

		renderer := html.NewRenderer(opts)

		markdownInput := rootTwt.FormatText(types.MarkdownFmt, conf)
		if subject, _ := GetTwtConvSubjectHash(cache, rootTwt); subject != "" {
			markdownInput = strings.ReplaceAll(markdownInput, "("+subject+")", "")
			markdownInput = strings.TrimSpace(markdownInput)
		}

		md := []byte(markdownInput)
		maybeUnsafeHTML := markdown.ToHTML(md, mdParser, renderer)

		p := bluemonday.UGCPolicy()
		p.AllowStandardURLs()
		// Override the allowed schemas as p.AllowStandardURLs only permits http, https and mailto
		p.AllowURLSchemes("mailto", "http", "https", "gemini", "gopher")
		p.AllowAttrs("id").OnElements("dialog")
		p.AllowAttrs("id", "controls").OnElements("audio")
		p.AllowAttrs("id", "controls", "playsinline", "preload", "poster").OnElements("video")
		p.AllowAttrs("src", "type").OnElements("source")
		p.AllowAttrs("aria-label", "class", "data-target", "target").OnElements("a")
		p.AllowAttrs("class", "data-target").OnElements("i", "lightbox")
		p.AllowAttrs("alt", "title", "loading", "data-target", "data-tooltip").OnElements("a", "img")
		html := p.SanitizeBytes(maybeUnsafeHTML)

		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
		if err != nil {
			log.WithError(err).Warn("error parsing twt context html")
			return template.HTML("")
		}

		firstParagraph, err := doc.Find("p").First().Html()
		if err != nil {
			log.WithError(err).Warn("error finding first paragraph for twt context")
			return template.HTML("")
		}

		return template.HTML(firstParagraph)
	}
}

// FormatMentionsAndTags turns `@<nick URL>` into `<a href="URL">@nick</a>`
// and `#<tag URL>` into `<a href="URL">#tag</a>` and a `!<hash URL>`
// into a `<a href="URL">!hash</a>`.
func FormatMentionsAndTags(conf *Config, text string, format TwtTextFormat) string {
	isLocalURL := IsLocalURLFactory(conf)
	re := regexp.MustCompile(`(@|#)<([^ ]+) *([^>]+)>`)
	return re.ReplaceAllStringFunc(text, func(match string) string {
		parts := re.FindStringSubmatch(match)
		prefix, nick, url := parts[1], parts[2], parts[3]

		if format == TextFmt {
			switch prefix {
			case "@":
				if isLocalURL(url) && strings.HasSuffix(url, "/twtxt.txt") {
					return fmt.Sprintf("%s@%s", nick, conf.baseURL.Hostname())
				}
				return fmt.Sprintf("@%s", nick)
			default:
				return fmt.Sprintf("%s%s", prefix, nick)
			}
		}

		if format == HTMLFmt {
			switch prefix {
			case "@":
				if isLocalURL(url) && strings.HasSuffix(url, "/twtxt.txt") {
					return fmt.Sprintf(`<a href="%s">@%s</a>`, UserURL(url), nick)
				}
				return fmt.Sprintf(`<a href="%s">@%s</a>`, URLForExternalProfile(conf, url), nick)
			default:
				return fmt.Sprintf(`<a href="%s">%s%s</a>`, url, prefix, nick)
			}
		}

		switch prefix {
		case "@":
			// Using (#) anchors to add the nick to URL for now. The Fluter app needs it since
			// 	the Markdown plugin doesn't include the link text that contains the nick in its onTap callback
			// https://github.com/flutter/flutter_markdown/issues/286
			return fmt.Sprintf(`[@%s](%s#%s)`, nick, url, nick)
		default:
			return fmt.Sprintf(`[%s%s](%s)`, prefix, nick, url)
		}
	})
}

// FormatRequest generates ascii representation of a request
func FormatRequest(r *http.Request) string {
	return fmt.Sprintf(
		"%s %v %s%v %v (%s)",
		r.RemoteAddr,
		r.Method,
		r.Host,
		r.URL,
		r.Proto,
		r.UserAgent(),
	)
}

func GetMediaNamesFromText(text string) []string {
	var mediaNames []string
	textSplit := strings.Split(text, "![](")
	for i, textSplitItem := range textSplit {
		if i > 0 {
			mediaEndIndex := strings.Index(textSplitItem, ")")
			if mediaEndIndex != -1 {
				mediaURL := textSplitItem[:mediaEndIndex]

				mediaURLSplit := strings.Split(mediaURL, "media/")
				for j, mediaURLSplitItem := range mediaURLSplit {
					if j > 0 {
						mediaPath := mediaURLSplitItem
						mediaNames = append(mediaNames, mediaPath)
					}
				}
			}
		}
	}
	return mediaNames
}

// NewMultiFeedLookup is used to chain together multiple feed lookup functions together
// where the first lookup that returns a non-zero Twter is used to expand the @-mention(2)
// before writing to the feed.
func NewMultiFeedLookup(feedLookupFns ...types.FeedLookup) types.FeedLookup {
	return types.FeedLookupFn(func(alias string) *types.Twter {
		for _, feedLookup := range feedLookupFns {
			if twter := feedLookup.FeedLookup(alias); !twter.IsZero() {
				return twter
			}
		}
		return &types.Twter{}
	})
}

// NewUserFollowedAsFeedLookup matches an @-mention based on the user's following list of
// feeds and the aliases used to follow those feeds.
func NewUserFollowedAsFeedLookup(user *User) types.FeedLookup {
	return types.FeedLookupFn(func(alias string) *types.Twter {
		for followedAs, followedURL := range user.Following {
			if strings.EqualFold(alias, followedAs) {
				u, err := url.Parse(followedURL)
				if err != nil {
					log.WithError(err).Warnf("error looking up follow alias %s for user %s", alias, user)
					return &types.Twter{}
				}
				parts := strings.SplitN(followedAs, "@", 2)

				if len(parts) == 2 && u.Hostname() == parts[1] {
					return &types.Twter{Nick: parts[0], URI: followedURL}
				}

				return &types.Twter{Nick: followedAs, URI: followedURL}
			}
		}

		return &types.Twter{}
	})
}

// NewCachedFeedLookup matches an @-mention based on cahced feeds in the cache
// and returns the first matching Twter that matches. this should be used as the
// last lookup function as there could be multiple feeds in the cache with similar
// Twter feed names.
func NewCachedFeedLookup(cache Cacher) types.FeedLookup {
	return types.FeedLookupFn(func(alias string) *types.Twter {
		twter := cache.FindTwter(alias)
		// If twter.URI has a / followed by a number at the end of the path, skip it
		// as this is an indicator of an archived feed and an otherwise invalid mention.
		if matched, err := regexp.MatchString(`\/\d+$`, twter.URI); err == nil && matched {
			return &types.Twter{}
		}
		return twter
	})
}

// NewLocalFeedLookup matches an @-mention based on a local feed on the current pod which
// may be another user or a user's fefed (sometimes called a persona). This should be used
// lower in the priority when used with a multi feed lookup.
func NewLocalFeedLookup(conf *Config, db Store) types.FeedLookup {
	return types.FeedLookupFn(func(alias string) *types.Twter {
		username := NormalizeUsername(alias)
		if db.HasUser(username) {
			return &types.Twter{Nick: username, URI: URLForUser(conf.BaseURL, username)}
		}

		return &types.Twter{}
	})
}

// NewRemoteFeedLookup matches an @-mention based on a remote lookup and should be the last
// lookup function used in a multi feed lookup as the operation is time consuming as it
// performe outbound network requests to resolve the @-metniond(s).
func NewRemoteFeedLookup(conf *Config) types.FeedLookup {
	return types.FeedLookupFn(func(alias string) *types.Twter {
		// TODO: Add context and timeout handling

		client := webfinger.NewClient(nil)
		client.AllowHTTP = true

		log.Debugf("performing webfinger lookup for: %s", alias)
		jrd, err := client.Lookup(strings.TrimPrefix(alias, "@"), nil)
		if err != nil {
			log.WithError(err).Warnf("error looking up %s via webfinger", alias)
			return &types.Twter{}
		}

		for _, link := range jrd.Links {
			log.Debugf("link rel=%s type=%s", link.Rel, link.Type)
			if link.Rel == webfinger.RelSelf {
				if link.Type == "text/plain" {
					log.Debugf("found link rel=%s type=%s", link.Rel, link.Type)
					return &types.Twter{Nick: alias, URI: link.Href}
				}
			}
		}

		return &types.Twter{}
	})
}

// TextWithEllipsis formats a a string with at most `maxLength` characters
// using an ellipsis (...) tto indicate more content...
func TextWithEllipsis(text string, maxLength int) string {
	if len(text) > maxLength {
		return fmt.Sprintf("%s ...", text[:maxLength])
	}
	return text
}

// MemoryUsage returns information about thememory used by the runtime
func MemoryUsage() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return fmt.Sprintf(
		"Alloc = %s TotalAlloc = %s Sys = %s NumGC = %d",
		humanize.Bytes(m.Alloc), humanize.Bytes(m.TotalAlloc),
		humanize.Bytes(m.Sys), m.NumGC,
	)
}

// validateOrCorrectTwt checks that a.Hash() matches a freshly
// made Twt (using fmt.Sprintf("%l", a) for the body). If it
// doesn’t, it tries stripping a trailing "/<digits>" from the
// URI and recomputes. Returns the corrected Twt if successful.
func validateOrCorrectTwt(a types.Twt) (types.Twt, error) {
	nick := a.Twter().Nick
	uri := a.Twter().URI
	ts := a.Created()
	text := fmt.Sprintf("%l", a)

	// 1) Recompute with original URI
	original := types.MakeTwt(types.NewTwter(nick, uri), ts, text)
	if a.Hash() == original.Hash() {
		return a, nil
	}

	// 2) Try stripping a trailing "/n" suffix (a potential pattern for archived feeds)
	cleanURI := stripNumericSuffix(uri)
	if cleanURI != uri {
		corrected := types.MakeTwt(types.NewTwter(nick, cleanURI), ts, text)
		if a.Hash() == corrected.Hash() {
			metrics.Counter("cache", "corrected_twts").Inc()
			return corrected, nil
		}
	}

	// Nothing worked
	return a, fmt.Errorf(
		"could not validate/correct Twt: got hash=%q, expected=%q",
		a.Hash(), original.Hash(),
	)
}

// stripNumericSuffix returns uri without a trailing "/<number>"
// if one exists, otherwise returns uri unmodified.
func stripNumericSuffix(uri string) string {
	if i := strings.LastIndex(uri, "/"); i != -1 {
		suffix := uri[i+1:]
		if _, err := strconv.Atoi(suffix); err == nil {
			return uri[:i]
		}
	}
	return uri
}
